package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadURLsFromFile_Text(t *testing.T) {
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "urls.txt")
	content := `# Comments should be ignored
https://example.com/file1.zip
https://example.com/file2.zip # inline comment
  https://example.com/file3.zip  
https://example.com/file1.zip/
`
	if err := os.WriteFile(txtFile, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	urls, err := ReadURLsFromFile(txtFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"https://example.com/file1.zip",
		"https://example.com/file2.zip",
		"https://example.com/file3.zip",
	}
	if !reflect.DeepEqual(urls, expected) {
		t.Errorf("got %v, want %v", urls, expected)
	}
}

func TestReadURLsFromFile_JSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Array of strings
	f1 := filepath.Join(tmpDir, "urls1.json")
	if err := os.WriteFile(f1, []byte(`["https://example.com/a.iso", "https://example.com/b.iso"]`), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	urls, err := ReadURLsFromFile(f1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 2 || urls[0] != "https://example.com/a.iso" {
		t.Errorf("urls = %v", urls)
	}

	// Array of objects
	f2 := filepath.Join(tmpDir, "urls2.json")
	if err := os.WriteFile(f2, []byte(`[{"url": "https://example.com/c.iso"}, {"link": "https://example.com/d.iso"}]`), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	urls2, err := ReadURLsFromFile(f2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls2) != 2 || urls2[1] != "https://example.com/d.iso" {
		t.Errorf("urls2 = %v", urls2)
	}

	// Root object with urls key
	f3 := filepath.Join(tmpDir, "urls3.json")
	if err := os.WriteFile(f3, []byte(`{"urls": ["https://example.com/e.iso", "https://example.com/f.iso"]}`), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	urls3, err := ReadURLsFromFile(f3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls3) != 2 || urls3[0] != "https://example.com/e.iso" {
		t.Errorf("urls3 = %v", urls3)
	}
}

func TestReadURLsFromFile_CSV(t *testing.T) {
	tmpDir := t.TempDir()

	// CSV with header
	f1 := filepath.Join(tmpDir, "batch.csv")
	content := `id,name,url
1,Alpha,https://example.com/alpha.tar.gz
2,Beta,https://example.com/beta.tar.gz
`
	if err := os.WriteFile(f1, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	urls, err := ReadURLsFromFile(f1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{
		"https://example.com/alpha.tar.gz",
		"https://example.com/beta.tar.gz",
	}
	if !reflect.DeepEqual(urls, expected) {
		t.Errorf("got %v, want %v", urls, expected)
	}

	// CSV without header
	f2 := filepath.Join(tmpDir, "noheader.csv")
	content2 := `foo,https://example.com/first.bin,bar
baz,https://example.com/second.bin,qux
`
	if err := os.WriteFile(f2, []byte(content2), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	urls2, err := ReadURLsFromFile(f2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls2) != 2 || urls2[0] != "https://example.com/first.bin" {
		t.Errorf("urls2 = %v", urls2)
	}
}
