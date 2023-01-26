package main

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestCBuildMerge(t *testing.T) {
	t.Run("we correctly merge slice fields", func(t *testing.T) {
		ff := &testingx.FakeFiller{}
		global := &cBuildEnv{}
		ff.Fill(global)
		local := &cBuildEnv{}
		ff.Fill(local)

		merged := cBuildMerge(global, local)
		typeinfo := reflect.TypeOf(merged).Elem()

		for idx := 0; idx < typeinfo.NumField(); idx++ {
			field := reflect.ValueOf(merged).Elem().Field(idx)
			switch got := field.Interface().(type) {
			case []string:
				gvalue := reflect.ValueOf(global).Elem().Field(idx).Interface().([]string)
				lvalue := reflect.ValueOf(local).Elem().Field(idx).Interface().([]string)
				expect := append([]string{}, gvalue...)
				expect = append(expect, lvalue...)
				if diff := cmp.Diff(expect, got); diff != "" {
					name := typeinfo.Field(idx).Name
					t.Errorf("field %s: expected %v, got %v", name, expect, got)
				}
			default:
				// nothing
			}
		}
	})

	t.Run("we correctly copy scalar fields", func(t *testing.T) {
		ff := &testingx.FakeFiller{}
		global := &cBuildEnv{}
		ff.Fill(global)
		local := &cBuildEnv{}
		ff.Fill(local)

		merged := cBuildMerge(global, local)
		typeinfo := reflect.TypeOf(merged).Elem()

		for idx := 0; idx < typeinfo.NumField(); idx++ {
			field := reflect.ValueOf(merged).Elem().Field(idx)
			switch got := field.Interface().(type) {
			case string:
				gvalue := reflect.ValueOf(global).Elem().Field(idx).Interface().(string)
				if diff := cmp.Diff(gvalue, got); diff != "" {
					name := typeinfo.Field(idx).Name
					t.Errorf("field %s: expected %v, got %v", name, gvalue, got)
				}
			default:
				// nothing
			}
		}
	})
}

func TestCBuildExportAutotools(t *testing.T) {
	global := &cBuildEnv{
		AR:       "ar",
		AS:       "gas",
		CC:       "clang",
		CFLAGS:   []string{"-Wall", "-Wextra"},
		CXX:      "clang++",
		CXXFLAGS: []string{"-Wall", "-Wextra", "-std=c++11"},
		LD:       "ld",
		LDFLAGS:  []string{"-L/usr/local/lib"},
		RANLIB:   "ranlib",
		STRIP:    "strip",
	}
	expect := &shellx.Envp{
		V: []string{
			"AR=ar",
			"AS=gas",
			"CC=clang",
			"CFLAGS=-Wall -Wextra",
			"CXX=clang++",
			"CXXFLAGS=-Wall -Wextra -std=c++11",
			"LD=ld",
			"LDFLAGS=-L/usr/local/lib",
			"RANLIB=ranlib",
			"STRIP=strip",
		},
	}
	got := cBuildExportAutotools(global)
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestCBuildExportOpenSSL(t *testing.T) {
	global := &cBuildEnv{
		ANDROID_HOME:     "/android",
		ANDROID_NDK_ROOT: "/android/sdk/ndk",
		CFLAGS:           []string{"-Wall", "-Wextra"},
		CXXFLAGS:         []string{"-Wall", "-Wextra", "-std=c++11"},
		LDFLAGS:          []string{"-L/usr/local/lib"},
	}
	expect := &shellx.Envp{
		V: []string{
			"ANDROID_HOME=/android",
			"ANDROID_NDK_HOME=/android/sdk/ndk",
			"CFLAGS=-Wall -Wextra",
			"CXXFLAGS=-Wall -Wextra -std=c++11",
			"LDFLAGS=-L/usr/local/lib",
		},
	}
	got := cBuildExportOpenSSL(global)
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Fatal(diff)
	}
}
