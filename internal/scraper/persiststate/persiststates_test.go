package persiststate

import (
	"log"
	"reflect"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func TestPersistSessionStates(t *testing.T) {
	t.Run("Testing if extracting session states work", func(t *testing.T) {
		browser, page := initialiseTestBrowser(t)
		defer browser.MustClose()
		defer page.MustClose()

		page.MustEval(`() => {
		sessionStorage.setItem("testSessionKey", "testSessionValue");
		localStorage.setItem("testLocalKey", "testLocalValue");
		}`)

		// Extract session storage and local storage data
		sessionStorage := ExtractStorageData(page, "sessionStorage")
		localStorage := ExtractStorageData(page, "localStorage")

		if len(sessionStorage) == 0 || len(localStorage) == 0 {
			log.Fatalf("Extracted JSON Map is empty")
		}

		// Verify the extracted data
		assertStorageEqual(t, sessionStorage["testSessionKey"], "testSessionValue")
		assertStorageEqual(t, localStorage["testLocalKey"], "testLocalValue")
	})
	t.Run("Testing if setting session states work", func(t *testing.T) {
		browser, page := initialiseTestBrowser(t)
		defer browser.MustClose()
		defer page.MustClose()

		cookies := []*proto.NetworkCookie{{Name: "TestName", Value: "TestValue", Domain: "TestDomain"}}
		sessionStorageSet := map[string]string{"testSessionStorageSet": "testSessionValueSet"}
		localStorageSet := map[string]string{"testLocalStorageSet": "testLocalStorageValueSet"}

		SetSessionData(page, cookies, sessionStorageSet, localStorageSet)
		// Extract session storage and local storage data
		sessionStorage := ExtractStorageData(page, "sessionStorage")
		localStorage := ExtractStorageData(page, "localStorage")

		// Verify the extracted data
		assertCookiesEqual(t, page.MustCookies(), cookies)
		assertStorageEqual(t, sessionStorageSet["testSessionStorageSet"], sessionStorage["testSessionStorageSet"])
		assertStorageEqual(t, localStorageSet["testLocalStorageSet"], localStorage["testLocalStorageSet"])

	})
}

func assertStorageEqual(t testing.TB, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func assertCookiesEqual(t testing.TB, got, want []*proto.NetworkCookie) {
	t.Helper()

	if reflect.DeepEqual(got, want) != true {
		log.Print("\nGot Cookie")
		for _, cookie := range got {
			log.Printf("Name: %s \nValue: %s, \nDomain: %s", cookie.Name, cookie.Value, cookie.Domain)
		}
		log.Print("\nWant Cookie")
		for _, cookie := range want {
			log.Printf("Want Cookie: \nName: %s \nValue: %s, \nDomain: %s", cookie.Name, cookie.Value, cookie.Domain)
		}

		log.Fatalf("Cookies did not set correctly")
	}
}

func initialiseTestBrowser(t testing.TB) (*rod.Browser, *rod.Page) {
	t.Helper()

	// Initialize the browser
	browser := rod.New().MustConnect()

	page := browser.MustPage()

	// Navigate to a test page that sets some session storage and local storage data
	page.MustNavigate("https://www.example.com").MustWaitLoad()

	return browser, page
}
