package main

import (
	"encoding/json"
	"testing"
)

func TestResourceJSON(t *testing.T) {
	statusCode := 200
	mbps := 10.5
	contentEncoding := ContentEncodingGzip

	resource := Resource{
		Method:          "GET",
		URL:             "https://example.com/api",
		TTFBMS:          150,
		StatusCode:      &statusCode,
		MBPS:            &mbps,
		ContentEncoding: &contentEncoding,
		RawHeaders:      HttpHeaders{"Content-Type": "application/json"},
	}

	// JSON エンコード
	jsonData, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	t.Logf("Resource JSON: %s", string(jsonData))

	// JSON デコード
	var decoded Resource
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	// 検証
	if decoded.Method != resource.Method {
		t.Errorf("Method mismatch: got %s, want %s", decoded.Method, resource.Method)
	}
	if decoded.URL != resource.URL {
		t.Errorf("URL mismatch: got %s, want %s", decoded.URL, resource.URL)
	}
	if *decoded.StatusCode != *resource.StatusCode {
		t.Errorf("StatusCode mismatch: got %d, want %d", *decoded.StatusCode, *resource.StatusCode)
	}
}


func TestInventoryJSON(t *testing.T) {
	entryURL := "https://example.com"
	deviceType := DeviceTypeDesktop

	inventory := Inventory{
		EntryURL:   &entryURL,
		DeviceType: &deviceType,
		Resources: []Resource{
			{Method: "GET", URL: "https://example.com/", TTFBMS: 100},
			{Method: "GET", URL: "https://example.com/style.css", TTFBMS: 50},
		},
	}

	// JSON エンコード
	jsonData, err := json.Marshal(inventory)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	t.Logf("Inventory JSON: %s", string(jsonData))

	// JSON デコード
	var decoded Inventory
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	// 検証
	if *decoded.EntryURL != *inventory.EntryURL {
		t.Errorf("EntryURL mismatch: got %s, want %s", *decoded.EntryURL, *inventory.EntryURL)
	}
	if len(decoded.Resources) != len(inventory.Resources) {
		t.Errorf("Resources length mismatch: got %d, want %d", len(decoded.Resources), len(inventory.Resources))
	}
}
