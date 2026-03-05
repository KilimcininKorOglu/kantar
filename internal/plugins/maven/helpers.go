// Package maven implements the Maven repository plugin for Kantar.
package maven

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// mavenPathInfo holds the parsed components of a Maven repository path.
type mavenPathInfo struct {
	groupID    string
	artifactID string
	version    string
	filename   string
	isMetadata bool
}

// parseMavenPath extracts groupId, artifactId, version, and filename from
// a Maven repository path. Maven paths follow the convention:
//
//	{groupPath}/{artifactId}/{version}/{filename}  — artifact
//	{groupPath}/{artifactId}/maven-metadata.xml    — metadata
//
// where groupPath is the groupId with dots replaced by slashes.
func parseMavenPath(rawPath string) (*mavenPathInfo, error) {
	path := strings.TrimPrefix(rawPath, "/")
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("path too short: need at least groupId/artifactId/version or metadata")
	}

	// Check if the last segment is maven-metadata.xml (metadata request).
	last := parts[len(parts)-1]
	if last == "maven-metadata.xml" {
		if len(parts) < 3 {
			return nil, fmt.Errorf("metadata path too short")
		}
		artifactID := parts[len(parts)-2]
		groupParts := parts[:len(parts)-2]
		groupID := strings.Join(groupParts, ".")

		return &mavenPathInfo{
			groupID:    groupID,
			artifactID: artifactID,
			isMetadata: true,
		}, nil
	}

	// Artifact path: {groupParts...}/{artifactId}/{version}/{filename}
	if len(parts) < 4 {
		return nil, fmt.Errorf("artifact path too short: need groupId/artifactId/version/filename")
	}

	filename := parts[len(parts)-1]
	version := parts[len(parts)-2]
	artifactID := parts[len(parts)-3]
	groupParts := parts[:len(parts)-3]
	groupID := strings.Join(groupParts, ".")

	if groupID == "" || artifactID == "" || version == "" || filename == "" {
		return nil, fmt.Errorf("invalid maven path: missing component")
	}

	return &mavenPathInfo{
		groupID:    groupID,
		artifactID: artifactID,
		version:    version,
		filename:   filename,
	}, nil
}

// mavenMetadata is the XML structure for maven-metadata.xml.
type mavenMetadata struct {
	XMLName    xml.Name          `xml:"metadata"`
	GroupID    string            `xml:"groupId"`
	ArtifactID string           `xml:"artifactId"`
	Versioning mavenVersioning  `xml:"versioning"`
}

// mavenVersioning holds the versioning block in maven-metadata.xml.
type mavenVersioning struct {
	Latest      string   `xml:"latest"`
	Release     string   `xml:"release"`
	Versions    versions `xml:"versions"`
	LastUpdated string   `xml:"lastUpdated"`
}

// versions wraps the list of version elements.
type versions struct {
	Version []string `xml:"version"`
}

// generateMetadataXML creates a maven-metadata.xml document for the given
// artifact with the provided version list.
func generateMetadataXML(groupID, artifactID string, versionList []string) ([]byte, error) {
	if len(versionList) == 0 {
		return nil, fmt.Errorf("no versions available")
	}

	sorted := make([]string, len(versionList))
	copy(sorted, versionList)
	sort.Strings(sorted)

	latest := sorted[len(sorted)-1]

	meta := mavenMetadata{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Versioning: mavenVersioning{
			Latest:      latest,
			Release:     latest,
			Versions:    versions{Version: sorted},
			LastUpdated: time.Now().UTC().Format("20060102150405"),
		},
	}

	return marshalXML(meta)
}

// marshalXML serializes a value to indented XML with the standard XML header.
func marshalXML(v any) ([]byte, error) {
	data, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling XML: %w", err)
	}
	return append([]byte(xml.Header), data...), nil
}

// writeXML writes XML content to the response with proper headers.
func writeXML(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write(data)
}

// writeErrorText writes a plain text error response.
func writeErrorText(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintln(w, msg)
}

// groupIDToPath converts a Maven groupId to a directory path
// (e.g., "com.example.lib" -> "com/example/lib").
func groupIDToPath(groupID string) string {
	return strings.ReplaceAll(groupID, ".", "/")
}
