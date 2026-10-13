package main

//
// check-okhttp: that OkHttp still configures Conscrypt sockets on the JVM as it does on Android.
//
// android.sh captures Signal-Android's hellos on the JVM with Conscrypt as the first security
// provider, so that OkHttp's JVM build uses ConscryptPlatform. That platform's
// configureTlsExtensions makes the same calls as the Android build's ConscryptSocketAdapter, which
// a phone uses for Signal's bundled Conscrypt (session tickets on, then ALPN). The two produced
// byte-identical hellos when checked against Android 14's own runtime (hellos/README.md). This
// compares both functions in the given OkHttp version's sources with the copies in
// okhttp-conscrypt-tls.txt, and fails if either changed, so that the equivalence is checked again
// before a capture is trusted.
//

import (
	"archive/zip"
	"bytes"
	"crypto/sha1" // #nosec G505 -- Maven Central publishes SHA-1 checksums
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// okhttpSources are the two functions compared: (artifact, path in its sources jar).
var okhttpSources = [][2]string{
	{"okhttp-android", "androidMain/okhttp3/internal/platform/android/ConscryptSocketAdapter.kt"},
	{"okhttp-jvm", "jvmMain/okhttp3/internal/platform/ConscryptPlatform.kt"},
}

// checkOkHTTP compares (or, with write, records) the functions of OkHttp version, downloading its
// sources jars into cache. reference is okhttp-conscrypt-tls.txt.
func checkOkHTTP(version, cache, reference string, write bool) error {
	var current strings.Builder
	for _, src := range okhttpSources {
		artifact, path := src[0], src[1]
		jar, err := sourcesJar(artifact, version, cache)
		if err != nil {
			return err
		}
		text, err := readZipFile(jar, path)
		if err != nil {
			return fmt.Errorf("%s: %w", artifact, err)
		}
		body := kotlinFunction(text, "configureTlsExtensions")
		if body == "" {
			return fmt.Errorf("no configureTlsExtensions in %s %s", artifact, path)
		}
		fmt.Fprintf(&current, "%s %s\n%s\n", artifact, path, body)
	}
	if write {
		header := fmt.Sprintf("# OkHttp %s: verified against Android 14's runtime (hellos/README.md)\n", version)
		if err := os.WriteFile(reference, []byte(header+current.String()), 0o644); err != nil {
			return err
		}
		fmt.Printf("check-okhttp: recorded OkHttp %s\n", version)
		return nil
	}
	data, err := os.ReadFile(reference)
	if err != nil {
		return err
	}
	var recorded strings.Builder
	for _, line := range strings.SplitAfter(string(data), "\n") {
		if !strings.HasPrefix(line, "#") {
			recorded.WriteString(line)
		}
	}
	if recorded.String() != current.String() {
		return fmt.Errorf("OkHttp %s changed how it configures Conscrypt sockets. The JVM capture may no longer "+
			"match a phone: check it against Android's runtime (hellos/README.md), then run check-okhttp --write", version)
	}
	fmt.Printf("check-okhttp: OkHttp %s configures Conscrypt sockets as verified\n", version)
	return nil
}

// sourcesJar returns the sources jar of an OkHttp artifact, downloaded into cache once and checked
// against Maven Central's SHA-1 every time.
// sourcesJar returns an OkHttp sources jar from the cache, or downloads it from Maven Central,
// checks it against Maven's SHA-1, and caches it.
func sourcesJar(artifact, version, cache string) ([]byte, error) {
	name := fmt.Sprintf("%s-%s-sources.jar", artifact, version)
	path := filepath.Join(cache, name)
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- our own cache
		return data, nil
	}
	url := fmt.Sprintf("https://repo1.maven.org/maven2/com/squareup/okhttp3/%s/%s/%s", artifact, version, name)
	data, err := download(url)
	if err != nil {
		return nil, err
	}
	want, err := download(url + ".sha1")
	if err != nil {
		return nil, err
	}
	sum := sha1.Sum(data) // #nosec G401 -- Maven Central's checksum
	if fields := strings.Fields(string(want)); len(fields) == 0 || hex.EncodeToString(sum[:]) != fields[0] {
		return nil, fmt.Errorf("%s does not match Maven's SHA-1", url)
	}
	if err := os.MkdirAll(cache, 0o750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path+".tmp", data, 0o600); err != nil {
		return nil, err
	}
	return data, os.Rename(path+".tmp", path)
}

var httpClient = &http.Client{Timeout: 2 * time.Minute}

func download(url string) ([]byte, error) {
	resp, err := httpClient.Get(url) // #nosec G107 -- Maven Central URLs built above
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<20))
}

func readZipFile(data []byte, name string) (string, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	f, err := z.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	text, err := io.ReadAll(f)
	return string(text), err
}

// kotlinFunction returns `override fun name(...) { ... }` from source with its whitespace
// collapsed, or "" when there is none.
func kotlinFunction(source, name string) string {
	m := regexp.MustCompile(`override fun ` + regexp.QuoteMeta(name) + `\(`).FindStringIndex(source)
	if m == nil {
		return ""
	}
	open := strings.Index(source[m[1]:], "{")
	if open < 0 {
		return ""
	}
	depth := 0
	for i := m[1] + open; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.Join(strings.Fields(source[m[0]:i+1]), " ")
			}
		}
	}
	return ""
}
