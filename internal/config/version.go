package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

// VersionInfo represents a GitHub tag
type VersionInfo struct {
	TagName string
	IsRC    bool
}

var versionCache []VersionInfo

// ResolveRefineryVersion resolves flexible version strings to a specific GitHub ref.
// Supports: latest, dev, nightly, major (2), major.minor (2.0), full semver (2.0.0)
func ResolveRefineryVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "latest", nil
	}

	// Handle special keywords
	switch strings.ToLower(version) {
	case "dev":
		return "main", nil
	case "nightly":
		tag, err := getLatestNightly()
		if err != nil {
			return "", fmt.Errorf("failed to get nightly version: %w", err)
		}
		return tag, nil
	case "latest":
		tag, err := getLatestStable()
		if err != nil {
			return "", fmt.Errorf("failed to get latest version: %w", err)
		}
		return tag, nil
	}

	// Check if it is a specific ref
	if strings.HasPrefix(version, "refs/") || len(version) == 40 {
		return version, nil
	}

	// Handle version patterns
	cleanVersion := strings.TrimPrefix(version, "v")
	parts := strings.Split(cleanVersion, ".")
	if len(parts) > 3 {
		return version, nil
	}

	versions, err := fetchVersions()
	if err != nil {
		return "", fmt.Errorf("failed to fetch versions: %w", err)
	}

	switch len(parts) {
	case 1:
		return findBestMatch(versions, parts[0], "", "")
	case 2:
		return findBestMatch(versions, parts[0], parts[1], "")
	case 3:
		return "v" + cleanVersion, nil
	}

	return version, nil
}

func fetchVersions() ([]VersionInfo, error) {
	if len(versionCache) > 0 {
		return versionCache, nil
	}

	// Get tags
	url := "https://api.github.com/repos/SirCesarium/refinery/tags?per_page=100"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	// Get releases for RC versions
	url = "https://api.github.com/repos/SirCesarium/refinery/releases?per_page=100"
	resp2, err := http.Get(url)
	if err == nil {
		defer resp2.Body.Close()
		var releases []struct {
			TagName string `json:"tag_name"`
		}
		if json.NewDecoder(resp2.Body).Decode(&releases) == nil {
			for _, r := range releases {
				tags = append(tags, struct {
					Name string `json:"name"`
				}{r.TagName})
			}
		}
	}

	for _, tag := range tags {
		name := strings.TrimPrefix(tag.Name, "v")
		isRC := strings.Contains(name, "-rc") || strings.Contains(name, "-beta")
		versionCache = append(versionCache, VersionInfo{
			TagName: "v" + name,
			IsRC:    isRC,
		})
	}

	return versionCache, nil
}

func getLatestStable() (string, error) {
	versions, err := fetchVersions()
	if err != nil {
		return "", err
	}

	var stable []VersionInfo
	for _, v := range versions {
		if !v.IsRC {
			stable = append(stable, v)
		}
	}

	if len(stable) == 0 {
		return "main", nil
	}

	sort.Slice(stable, func(i, j int) bool {
		return compareVersions(stable[i].TagName, stable[j].TagName) > 0
	})

	return stable[0].TagName, nil
}

func getLatestNightly() (string, error) {
	versions, err := fetchVersions()
	if err != nil {
		return "", err
	}

	var nightlies []VersionInfo
	for _, v := range versions {
		if v.IsRC {
			nightlies = append(nightlies, v)
		}
	}

	if len(nightlies) == 0 {
		return getLatestStable()
	}

	sort.Slice(nightlies, func(i, j int) bool {
		return compareVersions(nightlies[i].TagName, nightlies[j].TagName) > 0
	})

	return nightlies[0].TagName, nil
}

func findBestMatch(versions []VersionInfo, major, minor, patch string) (string, error) {
	pattern := fmt.Sprintf("^v%s", regexp.QuoteMeta(major))
	if minor != "" {
		pattern += fmt.Sprintf("\\.%s", regexp.QuoteMeta(minor))
		if patch != "" {
			pattern += fmt.Sprintf("\\.%s", regexp.QuoteMeta(patch))
		}
	}
	pattern += `(\..*)?$`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	var matches []VersionInfo
	for _, v := range versions {
		if re.MatchString(v.TagName) && !v.IsRC {
			matches = append(matches, v)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no version found matching pattern: %s", pattern)
	}

	sort.Slice(matches, func(i, j int) bool {
		return compareVersions(matches[i].TagName, matches[j].TagName) > 0
	})

	return matches[0].TagName, nil
}

func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		aPart := strings.Split(aParts[i], "-")[0]
		bPart := strings.Split(bParts[i], "-")[0]

		if aPart > bPart {
			return 1
		}
		if aPart < bPart {
			return -1
		}
	}

	if len(aParts) > len(bParts) {
		return 1
	}
	if len(aParts) < len(bParts) {
		return -1
	}
	return 0
}
