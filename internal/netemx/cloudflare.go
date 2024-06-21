package netemx

import (
	"log"
	"net"
	"net/http"

	"github.com/ooni/netem"
)

// CloudflareCAPTCHAHandlerFactory implements cloudflare CAPTCHAs.
func CloudflareCAPTCHAHandlerFactory() HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
		return CloudflareCAPTCHAHandler()
	})
}

var cloudflareCAPTCHAWebPage = []byte(`
<!DOCTYPE HTML>
<html lang="en-US">
<head>
  <meta charset="UTF-8" />
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
  <meta http-equiv="X-UA-Compatible" content="IE=Edge,chrome=1" />
  <meta name="robots" content="noindex, nofollow" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>Just a moment...</title>
  <style type="text/css">
    html, body {width: 100%; height: 100%; margin: 0; padding: 0;}
    body {background-color: #ffffff; color: #000000; font-family:-apple-system, system-ui, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, "Helvetica Neue",Arial, sans-serif; font-size: 16px; line-height: 1.7em;-webkit-font-smoothing: antialiased;}
    h1 { text-align: center; font-weight:700; margin: 16px 0; font-size: 32px; color:#000000; line-height: 1.25;}
    p {font-size: 20px; font-weight: 400; margin: 8px 0;}
    p, .attribution, {text-align: center;}
    #spinner {margin: 0 auto 30px auto; display: block;}
    .attribution {margin-top: 32px;}
    @keyframes fader     { 0% {opacity: 0.2;} 50% {opacity: 1.0;} 100% {opacity: 0.2;} }
    @-webkit-keyframes fader { 0% {opacity: 0.2;} 50% {opacity: 1.0;} 100% {opacity: 0.2;} }
    #cf-bubbles > .bubbles { animation: fader 1.6s infinite;}
    #cf-bubbles > .bubbles:nth-child(2) { animation-delay: .2s;}
    #cf-bubbles > .bubbles:nth-child(3) { animation-delay: .4s;}
    .bubbles { background-color: #f58220; width:20px; height: 20px; margin:2px; border-radius:100%; display:inline-block; }
    a { color: #2c7cb0; text-decoration: none; -moz-transition: color 0.15s ease; -o-transition: color 0.15s ease; -webkit-transition: color 0.15s ease; transition: color 0.15s ease; }
    a:hover{color: #f4a15d}
    .attribution{font-size: 16px; line-height: 1.5;}
    .ray_id{display: block; margin-top: 8px;}
    #cf-wrapper #challenge-form { padding-top:25px; padding-bottom:25px; }
    #cf-hcaptcha-container { text-align:center;}
    #cf-hcaptcha-container iframe { display: inline-block;}
  </style>

      <meta http-equiv="refresh" content="3">
  <script type="text/javascript">
    //<![CDATA[
    (function(){

      window._cf_chl_opt={
        cvId: "2",
        cType: "non-interactive",
        cNounce: "97691",
        cRay: "68013f3b0c7d498b",
        cHash: "c629373f5bf3ad9",
        cFPWv: "g",
        cTTimeMs: "1000",
        cRq: {
          ru: "aHR0cDovL2xnYnQuZm91bmRhdGlvbi8=",
          ra: "TW96aWxsYS81LjAgKFdpbmRvd3MgTlQgMTAuMDsgV2luNjQ7IHg2NCkgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgQ2hyb21lLzkwLjAuNDQzMC45MyBTYWZhcmkvNTM3LjM2",
          rm: "R0VU",
          d: "pQ3nb5AcDwNb7z9ijyLMXcEOanHjT9r1ePf6pKvwmBOwOZaUr9bWr1Kgh9OBvMNycB+8tQDFF0f9/X7spkTKj26ep0JXAHgO8oG/W0INjw0cX9w7WEBNtt/PsjtOiaRZZsC+2P0w/bF1zy3VNKqyMGQFWJt3Lky+6pVbtVTTIhvsEAwstJ9MINQfdVg9wOJx0r95PXHJGg8y4jhIeF2gAqMza9Ug5iG0DHy7BBPG63MgJtyxydzETS5d59QcU1vVyzsRJBV/qWFl7J8CMJnUC2ox0ObDCeYZFrD+GSzcUGTqV/lV0mEM/SaARusI3OO4o/H74cgrN4H509yr9R9br1EuVbD/6NRQsc8zX5GsDYxKRJVscjTqS9c1pTxjs3/lPLJu6jTIX6itJlHjIGTYem1jB6PWsl4t0eln4VQPuOYzcUf/wMQ71vNLIl8M0XS496ya+ZMWOu3NUMs78j75JshdOkYfZ5WQx5i6hafo+0Kq2eACqLQ5SLMvt/Wn9YxG2QpVmJcnxZ6qV+2kN4MSOCzDEihEgMDsvbT2xObsG33QG4CRWqlM1z4uF4ZC583gaRrHVznUAeSwExSb2rIMPSAE64vYguOiim1d/n9LcEpXoccZkVNGo3jycEx53PVLWtZSMnbRKfDcYkZ/Fs91nf8F0o9Bb8u0E/Dd24NrBtxv5I7COi9WanIYGL7udFvepGhNfYjvfFAeXkg+1SRa74FGtsPU5KkfsHomvWLnl6kZh84u1RV9bsl5TB1ikWeBMFW65U3auWcsZ67vx4d06cZZOdrj/lQzp3G+P2wDuOY=",
          t: "MTYyOTE4NTk0OS45MzQwMDA=",
          m: "u/uh23t85V0m+MUzBn5qsZusVWzbP8zUDIdN8CSbST0=",
          i1: "aPsCmoX67SE1bs3V7UxfEw==",
          i2: "Dj31pDsLDJkIqJuXiVMXFw==",
          zh: "MOAGc57RydtNLJSEH/prEsUTDPR9h3Jow/mE27NBkek=",
          uh: "RhRfQ3Y9htiPyeXJ4MFfcXRnFwYa8lIqcK50BCUW5Uc=",
          hh: "GESBVfXNbuLcsV0d0Da/Xo4pc7AtEpsiRnBGcvFMGtc=",
        }
      }
      window._cf_chl_enter = function(){window._cf_chl_opt.p=1};

    })();
    //]]>
  </script>


</head>
<body>
  <table width="100%" height="100%" cellpadding="20">
    <tr>
      <td align="center" valign="middle">
          <div class="cf-browser-verification cf-im-under-attack">
  <noscript>
    <h1 data-translate="turn_on_js" style="color:#bd2426;">Please turn JavaScript on and reload the page.</h1>
  </noscript>
  <div id="cf-content" style="display:none">

    <div id="cf-bubbles">
      <div class="bubbles"></div>
      <div class="bubbles"></div>
      <div class="bubbles"></div>
    </div>
    <h1><span data-translate="checking_browser">Checking your browser before accessing</span> lgbt.foundation.</h1>
    <!-- <a href="http://lagungroen.com/telephonequinquenni.php?source=415">table</a> -->
    <div id="no-cookie-warning" class="cookie-warning" data-translate="turn_on_cookies" style="display:none">
      <p data-translate="turn_on_cookies" style="color:#bd2426;">Please enable Cookies and reload the page.</p>
    </div>
    <p data-translate="process_is_automatic">This process is automatic. Your browser will redirect to your requested content shortly.</p>
    <p data-translate="allow_5_secs" id="cf-spinner-allow-5-secs" >Please allow up to 5 seconds&hellip;</p>
    <p data-translate="redirecting" id="cf-spinner-redirecting" style="display:none">Redirecting&hellip;</p>
  </div>

  <form class="challenge-form" id="challenge-form" action="/?__cf_chl_jschl_tk__=pmd_ebe1a9dbe4472c291e4c0b91fa2baf24a8e8ef05-1629185949-0-gqNtZGzNAc2jcnBszQLi" method="POST" enctype="application/x-www-form-urlencoded">
    <input type="hidden" name="md" value="b119c08bd231a98dcce3bae40d5228171a3689b4-1629185949-0-Ac9eUX6TZjGVopJgxPXVUTUAe0F-gV3bf1HzoF8kvKWVPajao7kLQnzH3ovGAEIMtS4bFWvYJNePC8xo1DkjCNmWTvUYf8ZcRD5RNytBkpT-8QE4YCQcHlwe7ePAgMvwbzCCQmHKk0-LSez1O8KAN6waDO5FQ6lgNoyvfXUbHqaqO-K7bd4toaFfPg7KKCl1TdWRP1FvvLM6bR6YvRtDIV7vIS9cXUPkqY-ETf2M89hCxxz3pLeHPkbY1wUBBNqVZkbUXgKXsBXwVBnON9cpN0wZE_XG_xVnKGJIlwpw3BVcigabXr2HuRlOLWhoOEWOol1Ex4iOFCuYn5Si69ANEWHDeIBDKi4VaL1yI83s6mT0V_mNVnXIPJumq3Rf1cw8Nhrh5yC02x-5oAzH3Q0VY2k" />
    <input type="hidden" name="r" value="8c5d3a196c30c0e49d1c11064661bb15d94cf3d6-1629185949-0-Adi/o7JWFCfIjUMhJICUSNCO3cAf/JIKgyf7eAPHZ4fF+06SJrzHqPHFwyCMU7BZVC7YyWKz9fUhqrCGg4PzhQBa4j5x0EKKsppeYQASYIGRNOv6JUaFey7WF0tF9u60XfO8Oxf/wrfZ8SF+Q+zjQcAZ1XXFW+wBMCipw6lXtsb86VvawP/ZD1OfVsCKGXcDiW65W7JUqIeNuToVleOtWBdkwpPlZeTUk/wsjgMnovNw2icXi1rXigjv8SP6BA+6HUM5a/tpuSpD1vcG0TfyNsP10+0PfyVynd3S2spXqJ9lddzbIh0dz5IsZVaaZsNu7FTFWC5y99USlacy38t2zwiGtUKhlDM/DIYH8QcDuKnA7/Crn+atOOiYojNjB2qFJnMBsX1sWJf65XeTBWTy7VYHj2CYqaxKgCllf//NuNY9s6W71ZRLoGGjHndGfJV06u3ILgR2gJIzY59wdktHCQDRqvw3vSe4eO/GWHRjtBBF9pRJHzOtDDHJSitoR2CxssYp+eBEMy+C/0q9ntdgzZ/0UeANQXlgku/az5hSYWM8krlDg+dWERBFLoUqg6B83gfslCvLsYwt08bpZvOnrdWWHxRX8YZCEZXxmiytmIZBXH3ucsC/b+pLH/R56U4PHCQomW+so1HaBpdPsnt90BHo+xUIWbBBwbRjawAkTRz8"/>
    <input type="hidden" value="c6ad5d3b5ede5b9831aa8584aadd6b50" id="jschl-vc" name="jschl_vc"/>
    <!-- <input type="hidden" value="" id="jschl-vc" name="jschl_vc"/> -->
    <input type="hidden" name="pass" value="1629185950.934-L7Wyj487wQ"/>
    <input type="hidden" id="jschl-answer" name="jschl_answer"/>
  </form>

    <script type="text/javascript">
      //<![CDATA[
      (function(){
          var a = document.getElementById('cf-content');
          a.style.display = 'block';
          var isIE = /(MSIE|Trident\/|Edge\/)/i.test(window.navigator.userAgent);
          var trkjs = isIE ? new Image() : document.createElement('img');
          trkjs.setAttribute("src", "/cdn-cgi/images/trace/jschal/js/transparent.gif?ray=68013f3b0c7d498b");
          trkjs.id = "trk_jschal_js";
          trkjs.setAttribute("alt", "");
          document.body.appendChild(trkjs);
          var cpo=document.createElement('script');
          cpo.type='text/javascript';
          cpo.src="/cdn-cgi/challenge-platform/h/g/orchestrate/jsch/v1?ray=68013f3b0c7d498b";
          document.getElementsByTagName('head')[0].appendChild(cpo);
        }());
      //]]>
    </script>

  <div id="trk_jschal_nojs" style="background-image:url('/cdn-cgi/images/trace/jschal/nojs/transparent.gif?ray=68013f3b0c7d498b')"> </div>
</div>


          <div class="attribution">
            DDoS protection by <a rel="noopener noreferrer" href="https://www.cloudflare.com/5xx-error-landing/" target="_blank">Cloudflare</a>
            <br />
            <span class="ray_id">Ray ID: <code>68013f3b0c7d498b</code></span>
          </div>
      </td>

    </tr>
  </table>
</body>
</html>
`)

// CloudflareCAPTCHAHandler returns the [http.Handler] for cloudflare CAPTCHAs. This handler
// returns the cloudflare CAPTCHA if the client address equals [DefaultClientAddress] and returns
// the [ExampleWebPage] otherwise. Therefore, we're modeling a cloudflare cache considering the
// client as untrusted and the test helper as trusted.
func CloudflareCAPTCHAHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Alt-Svc", `h3=":443"`)
		w.Header().Add(
			"Cache-Control",
			"private, max-age=0, no-store, no-cache, must-revalidate, post-check=0, pre-check=0",
		)
		w.Header().Add("Cf-Ray", "68013f3b0c7d498b-SIN")
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")
		w.Header().Add("Expires", "Thu, 01 Jan 1970 00:00:01 GMT")
		w.Header().Add("Nel", `{"success_fraction":0,"report_to":"cf-nel","max_age":604800}`)
		w.Header().Add(
			"Permissions-Policy",
			`accelerometer=(),autoplay=(),camera=(),clipboard-read=(),clipboard-write=(),fullscreen=(),geolocation=(),gyroscope=(),hid=(),interest-cohort=(),magnetometer=(),microphone=(),payment=(),publickey-credentials-get=(),screen-wake-lock=(),serial=(),sync-xhr=(),usb=()`,
		)
		w.Header().Add(
			"Report-To",
			`{"endpoints":[{"url":"https:\/\/a.nel.cloudflare.com\/report\/v3?s=9SevtxfJtcGPMEGxphEr1sQHmEGpEnsQ5W4Qhy6ns8aRqSPQm%2BeiRdEd05hO4THbNXuGnE9Wb0TFofv60U%2FFDA9P5eCqihYMH%2Bd36I0f%2BJXuVRVnZaH5ANSv4LZou%2FbxGQs%3D"}],"group":"cf-nel","max_age":604800}`,
		)
		w.Header().Add("Server", "cloudflare")
		w.Header().Add("X-Frame-Options", "SAMEORIG")

		// missing address => 500
		address, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("CLOUDFLARE_CACHE: missing address in request => 500")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// client => 503
		if address == DefaultClientAddress {
			log.Printf("CLOUDFLARE_CACHE: request from %s => 503", address)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(cloudflareCAPTCHAWebPage)
			return

		}

		// otherwise => 200
		log.Printf("CLOUDFLARE_CACHE: request from %s => 200", address)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ExampleWebPage))
	})
}
