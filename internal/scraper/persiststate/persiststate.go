package persiststate

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type StorageData map[string]string

func ExtractStorageData(page *rod.Page, storageType string) StorageData {
	data := make(map[string]string)
	storage := page.MustEval(fmt.Sprintf(`() => {
		const data = {};
		for (let i = 0; i < %s.length; i++) {
			const key = %s.key(i);
			data[key] = %s.getItem(key);
		}
		return data;
	}`, storageType, storageType, storageType))

	storage.Unmarshal(&data)

	return data

	/*
		for key, value := range storage {
			data[key] = value.(string)
		}
		return data */
}

func SetSessionData(page *rod.Page, cookies []*proto.NetworkCookie, sessionStorage, localStorage map[string]string) {
	page.MustSetCookies(proto.CookiesToParams(cookies)...)

	page.MustEval(`(data) => {
		for (const [key, value] of Object.entries(data)) {
			sessionStorage.setItem(key, value);
		}
	}`, sessionStorage)

	page.MustEval(`(data) => {
		for (const [key, value] of Object.entries(data)) {
			localStorage.setItem(key, value);
		}
	}`, localStorage)
}
