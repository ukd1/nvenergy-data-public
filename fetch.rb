require 'net/http'
require 'json'
require 'openssl'
require 'uri'
require 'nokogiri'
require 'date'
require 'csv'

ACCESS_TOKEN = JSON.load_file('token.json')['access_token'].freeze

def fahrenheit_to_celsius(fahrenheit)
  ((fahrenheit - 32) * 5.0 / 9.0).round(2) unless fahrenheit.nil?
end

def inch_to_mm(inch)
  inch * 25.4 unless inch.nil?
end

def txt_to_f(txt)
  txt&.split(" ")&.first.to_f unless txt.nil?
end


def extract_weather_data(weather_html)
  doc = Nokogiri::HTML::DocumentFragment.parse(weather_html.to_s)
  {
    generalWeather: doc.css('span[style="font-size:18px"]')[0]&.text&.strip,
    temperatureCelsius: fahrenheit_to_celsius(txt_to_f(doc.css('span[style="font-size:18px"]')[1]&.text)),
    humidityPercent: txt_to_f(doc.css('span[style="font-size:18px"]')[2]&.text),
    windSpeedMph: txt_to_f(doc.css('span[style="font-size:18px"]')[3]&.text),
    precipitationMm: inch_to_mm(txt_to_f(doc.css('span[style="font-size:18px"]')[4]&.text))
  }
rescue => e
  raise e
end

def fetch(dt)
  headers = {
    'Accept' => 'application/json, text/plain, application/com.nvenergy',
    'Content-Type' => 'application/json',
    'Authorization' => ACCESS_TOKEN,
    'X-CDX-CUSTOM-DATA' => 'B2C'
  }

  print "Fetching data for #{dt} "

  body = {
    "interval" => "15min",
    "meterNumber" => ENV['NV_ENERGY_METER_NUMBER'],
    "endDate" => dt,
    "isRequestForTile" => false,
    "hideDetailInNet" => false,
    "chartSelection" => "DEFAULT",
    "billType" => "G",
    "nvesource" => "CUSTOMER WEB ACCESS(CWA)",
    "userAccountNumber" => ENV['NV_ENERGY_ACCOUNT_NUMBER']
  }.to_json

  uri = URI('https://services.nvenergy.com/api/1.0/cdx/viewusage/getChartData')
  http = Net::HTTP.new(uri.host, uri.port)
  http.use_ssl = true
  http.verify_mode = OpenSSL::SSL::VERIFY_NONE # Not recommended for production

  request = Net::HTTP::Post.new(uri, headers)
  request.body = body
  response = http.request(request)

  if response.code == '200'
    rows = JSON.parse(response.body.lines[1..].join("\n"))
    # File.write("data/#{dt}.json", JSON.pretty_generate(rows))

    if !rows['isSuccess']
      raise 'Failed to fetch data - api returned unsuccessful.'
    elsif !rows['ResponseBody']['chart']
      puts 'No data (yet?) 🤷🏻'
    else
      rows = rows['ResponseBody']['chart']['dataProvider'].map(&:to_h)
      keys = (rows.first.keys - %W{kwhBalloonText tempBaloonText}) + extract_weather_data(nil).keys

      CSV.open("data/#{dt}.csv", 'wb') do |csv|
        csv << keys

        rows.each do |row|
          row.merge!(extract_weather_data(row['tempBaloonText']))

          csv << keys.map { |key| row[key] }
        end
      end

      puts '✅'
    end
  else
    puts response.body.inspect
    puts "🤦🏻‍♂️ Got bad response #{response.code}."
    exit 0
  end
end

def check(dts)
  CSV.open("data/#{dts}.csv", "r+", headers: false) do |csv|
    rows = csv.read
    keys = (rows.first + extract_weather_data(nil).keys).map(&:to_s).uniq

    if rows[0] != keys
      rows[0] = keys
    end

    rows.map! do |row|
      if row.length < keys.length
        row.fill(nil, row.length...keys.length)
      else
        row  # If the row has the correct number of elements, use it as is
      end
    end

    # Rewrite the file with adjusted rows (optional)
    CSV.open("data/#{dts}.csv", "wb") do |csv_out|
      rows.each { |row| csv_out << row }
    end
  end
end

(Date.new(2021, 7, 17)...(Date.today - 1)).each do |dt|
# (Date.new(2021, 7, 17)...Date.new(2021, 7, 20)).each do |dt|
  unless File.exist?("data/#{dt}.csv")
    fetch(dt.to_s)
  else
    check(dt.to_s)
    puts "Already got #{dt} 😇"
  end
end
