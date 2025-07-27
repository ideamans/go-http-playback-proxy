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
		TTFBMs:          150,
		StatusCode:      &statusCode,
		Mbps:            &mbps,
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

func TestDomainJSON(t *testing.T) {
	domain := Domain{
		Name:      "example.com",
		IPAddress: "192.168.1.1",
	}

	// JSON エンコード
	jsonData, err := json.Marshal(domain)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	t.Logf("Domain JSON: %s", string(jsonData))

	// JSON デコード
	var decoded Domain
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	// 検証
	if decoded.Name != domain.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, domain.Name)
	}
	if decoded.IPAddress != domain.IPAddress {
		t.Errorf("IPAddress mismatch: got %s, want %s", decoded.IPAddress, domain.IPAddress)
	}
}

func TestInventoryJSON(t *testing.T) {
	entryURL := "https://example.com"
	deviceType := DeviceTypeDesktop
	
	inventory := Inventory{
		EntryURL:   &entryURL,
		DeviceType: &deviceType,
		Domains: []Domain{
			{Name: "example.com", IPAddress: "192.168.1.1"},
			{Name: "cdn.example.com", IPAddress: "192.168.1.2"},
		},
		Resources: []Resource{
			{Method: "GET", URL: "https://example.com/", TTFBMs: 100},
			{Method: "GET", URL: "https://example.com/style.css", TTFBMs: 50},
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
	if len(decoded.Domains) != len(inventory.Domains) {
		t.Errorf("Domains length mismatch: got %d, want %d", len(decoded.Domains), len(inventory.Domains))
	}
	if len(decoded.Resources) != len(inventory.Resources) {
		t.Errorf("Resources length mismatch: got %d, want %d", len(decoded.Resources), len(inventory.Resources))
	}
}

