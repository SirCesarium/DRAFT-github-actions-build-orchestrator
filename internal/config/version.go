package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// ResolveRefineryVersion resolves flexible version strings to a GitHub ref.
// Supports: latest, dev, nightly, major (2), major.minor (2.0), full semver (2.0.0).
func ResolveRefineryVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "latest"
	}

	// Handle special keywords that need API calls
	switch strings.ToLower(version) {
	case "dev":
		return "main", nil
	case "nightly":
		return getLatestNightly()
	case "latest":
		return getLatestStable()
	}

	// Check if it's a specific ref (refs/ or commit hash)
	if strings.HasPrefix(version, "refs/") || len(version) == 40 {
		return version, nil
	}

	// For version patterns, resolve via API
	cleanVersion := strings.TrimPrefix(version, "v")
	parts := strings.Split(cleanVersion, ".")

	// For major (2) or major.minor (2.0), find latest matching version via API
	if len(parts) <= 2 {
		return findLatestMatching(cleanVersion)
	}

	// For full semver (2.0.0), return as-is
	return version, nil
}

// fetchTags gets all tags from GitHub API
func fetchTags() ([]string, error) {
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

	var tagNames []string
	for _, t := range tags {
		tagNames = append(tagNames, t.Name)
	}
	return tagNames, nil
}

// fetchReleases gets all releases (for RC versions)
func fetchReleases() ([]string, error) {
	url := "https://api.github.com/repos/SirCesarium/refinery/releases?per_page=100"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	var tagNames []string
	for _, r := range releases {
		tagNames = append(tagNames, r.TagName)
	}
	return tagNames, nil
}

// getLatestStable returns the latest non-RC version
func getLatestStable() (string, error) {
	tags, err := fetchTags()
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags: %w", err)
	}

	var stable []string
	for _, tag := range tags {
		name := strings.TrimPrefix(tag, "v")
		if !strings.Contains(name, "-rc") && !strings.Contains(name, "-beta") {
			stable = append(stable, tag)
		}
	}

	if len(stable) == 0 {
		return "main", nil
	}
	return findHighestVersion(stable), nil
}

// getLatestNightly returns the latest RC or beta version
func getLatestNightly() (string, error) {
	tags, err := fetchTags()
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags: %w", err)
	}
	releases, err := fetchReleases()
	if err == nil {
		tags = append(tags, releases...)
	}

	var nightlies []string
	for _, tag := range tags {
		name := strings.TrimPrefix(tag, "v")
		if strings.Contains(name, "-rc") || strings.Contains(name, "-beta") {
			nightlies = append(nightlies, tag)
		}
	}

	if len(nightlies) == 0 {
		return getLatestStable()
	}

	return findHighestVersion(nightlies), nil
}

// findHighestVersion returns the highest version from a list of version strings
func findHighestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	highest := versions[0]
	for _, v := range versions[1:] {
		if compareVersions(v, highest) > 0 {
			highest = v
		}
	}
	return highest
}

// compareVersions compares two version strings semantically (v2.0.10 > v2.0.9)
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	aParts := strings.Split(a, "-")[0] // Remove -rc suffix
	bParts := strings.Split(b, "-")[0]

	aNums := strings.Split(aParts, ".")
	bNums := strings.Split(bParts, ".")

	for i := 0; i < len(aNums) && i < len(bNums); i++ {
		aNum := atoi(aNums[i])
		bNum := atoi(bNums[i])
		if aNum > bNum {
			return 1
		}
		if aNum < bNum {
			return -1
		}
	}

	if len(aNums) > len(bNums) {
		return 1
	}
	if len(aNums) < len(bNums) {
		return -1
	}
	return 0
}

// atoi converts string to int, returns 0 if invalid
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

// findLatestMatching finds the latest version matching a prefix (e.g., "2" finds latest v2.x.x)
func findLatestMatching(prefix string) (string, error) {
	tags, err := fetchTags()
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags: %w", err)
	}

	releases, err := fetchReleases()
	if err == nil {
		tags = append(tags, releases...)
	}

	// Build pattern: ^v<major>(\.<minor>(\.\d+)?)?$
	pattern := fmt.Sprintf(`^v%s(\.\d+)?(\.\d+)?$`, regexp.QuoteMeta(prefix))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	var matches []string
	for _, tag := range tags {
		if re.MatchString(tag) {
			// Exclude RC/beta for stable versions
			name := strings.TrimPrefix(tag, "v")
			if !strings.Contains(name, "-rc") && !strings.Contains(name, "-beta") {
				matches = append(matches, tag)
			}
		}
	}

	if len(matches) == 0 {
		// Fall back to returning the prefix with v
		return "v" + prefix, nil
	}

	return findHighestVersion(matches), nil
}
