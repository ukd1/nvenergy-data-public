require 'ferrum'
require 'httparty'
require 'json'

browser = Ferrum::Browser.new(browser_options: { "headless": "new" }, timeout: 30)
browser.network.intercept
browser.on(:request) do |request|
  if request.match?(/bheb2c\.onmicrosoft\.com\/oauth2\/v2\.0\/token/)
    x = HTTParty.post(request.url, headers: request.headers, body: request.body)
    token = JSON.parse(x.body)
    File.write("token.json", JSON.pretty_generate(token))

    puts "🫡"
    exit 0
  else
    request.continue
    # puts request.url
    print "."
  end
end

browser.go_to("https://www.nvenergy.com/my-account")

while browser.at_css('input#signInName').nil? || browser.at_css('input#password').nil? || browser.at_css('button[type="submit"]').nil?
  sleep 0.5
end
sleep 2

browser.at_css('input#signInName').focus.type(ENV["NV_ENERGY_USER"])
browser.at_css('input#password').focus.type(ENV["NV_ENERGY_PASS"])
browser.at_css('button[type="submit"]').click
browser.network.wait_for_idle

browser.quit
