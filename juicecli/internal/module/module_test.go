package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetPackageName_ValidPackage(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file.Close()
	_, err = file.WriteString("package main\n")
	if err != nil {
		t.Fatalf("failed to write to test file: %v", err)
	}

	packageName, err := GetPackageName(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if packageName != "main" {
		t.Errorf("expected package name 'main', got '%s'", packageName)
	}
}

func TestGetPackageName_NoGoFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := GetPackageName(dir)
	if err == nil || err.Error() != "can not find package name" {
		t.Errorf("expected error 'can not find package name', got '%v'", err)
	}
}

func TestGetPackageName_InvalidPackage(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file.Close()
	_, err = file.WriteString("invalid content\n")
	if err != nil {
		t.Fatalf("failed to write to test file: %v", err)
	}

	_, err = GetPackageName(dir)
	if err == nil || err.Error() != "can not find package name" {
		t.Errorf("expected error 'can not find package name', got '%v'", err)
	}
}

func TestGetPackageName_MultipleGoFiles(t *testing.T) {
	dir := t.TempDir()
	file1, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file1.Close()
	_, err = file1.WriteString("package main\n")
	if err != nil {
		t.Fatalf("failed to write to test file: %v", err)
	}

	file2, err := os.Create(filepath.Join(dir, "utils.go"))
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file2.Close()
	_, err = file2.WriteString("package utils\n")
	if err != nil {
		t.Fatalf("failed to write to test file: %v", err)
	}

	packageName, err := GetPackageName(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if packageName != "main" {
		t.Errorf("expected package name 'main', got '%s'", packageName)
	}
}
