package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

var (
	outputDir = flag.String("output-dir", "manifest_staging/charts/workload-identity-webhook", "The root directory in which to write the Helm chart")
)

var kindRegex = regexp.MustCompile(`(?m)^kind:[\s]+([\S]+)[\s]*$`)

// use exactly two spaces to be sure we are capturing metadata.name
var nameRegex = regexp.MustCompile(`(?m)^  name:[\s]+([\S]+)[\s]*$`)

func extractKind(s string) (string, error) {
	matches := kindRegex.FindStringSubmatch(s)
	if len(matches) != 2 {
		return "", fmt.Errorf("%s does not have a kind", s)
	}
	return strings.Trim(matches[1], `"'`), nil
}

func extractName(s string) (string, error) {
	matches := nameRegex.FindStringSubmatch(s)
	if len(matches) != 2 {
		return "", fmt.Errorf("%s does not have a name", s)
	}
	return strings.Trim(matches[1], `"'`), nil
}

type kindSet struct {
	byKind map[string][]string
}

func (ks *kindSet) Add(obj string) error {
	kind, err := extractKind(obj)
	if err != nil {
		return err
	}
	objs, ok := ks.byKind[kind]
	if !ok {
		objs = []string{obj}
	} else {
		objs = append(objs, obj)
	}
	ks.byKind[kind] = objs
	return nil
}

func (ks *kindSet) Write() error {
	for kind, objs := range ks.byKind {
		subPath := "templates"
		nameExtractor := extractName

		if kind == "Namespace" {
			continue
		}

		for _, obj := range objs {
			name, err := nameExtractor(obj)
			if err != nil {
				return err
			}
			fileName := fmt.Sprintf("%s-%s.yaml", strings.ToLower(name), strings.ToLower(kind))
			destFile := path.Join(*outputDir, subPath, fileName)
			fmt.Printf("Writing %s\n", destFile)

			if kind == "Deployment" {
				obj = strings.Replace(obj, "      labels:", "      labels:\n{{- include \"workload-identity-webhook.podLabels\" . }}", 1)
			}

			if err := os.WriteFile(destFile, []byte(obj), 0600); err != nil {
				return err
			}
		}
	}
	return nil
}

func doReplacements(obj string) string {
	for old, new := range replacements {
		obj = strings.ReplaceAll(obj, old, new)
	}
	return obj
}

func copyStaticFiles(root string, subdirs ...string) error {
	p := path.Join(append([]string{root}, subdirs...)...)
	files, err := os.ReadDir(p)
	if err != nil {
		return err
	}
	for _, f := range files {
		newSubDirs := append([]string{}, subdirs...)
		newSubDirs = append(newSubDirs, f.Name())
		destination := path.Join(append([]string{*outputDir}, newSubDirs...)...)
		if f.IsDir() {
			fmt.Printf("Making %s\n", destination)
			if err := os.Mkdir(destination, 0750); err != nil {
				return err
			}
			if err := copyStaticFiles(root, newSubDirs...); err != nil {
				return err
			}
		} else {
			contents, err := os.ReadFile(path.Join(p, f.Name())) // #nosec G304
			if err != nil {
				return err
			}
			fmt.Printf("Writing %s\n", destination)
			if err := os.WriteFile(destination, contents, 0600); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	scanner := bufio.NewScanner(os.Stdin)
	kinds := kindSet{byKind: make(map[string][]string)}
	b := strings.Builder{}
	notate := func() {
		obj := doReplacements(b.String())
		b.Reset()
		if err := kinds.Add(obj); err != nil {
			log.Fatalf("Error adding object: %s, %s", err, b.String())
		}
	}

	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "---") {
			if b.Len() > 0 {
				notate()
			}
		} else {
			b.WriteString(scanner.Text())
			b.WriteString("\n")
		}
	}
	if b.Len() > 0 {
		notate()
	}
	if err := copyStaticFiles("third_party/open-policy-agent/gatekeeper/helmify/static"); err != nil {
		log.Fatal(err)
	}
	if err := kinds.Write(); err != nil {
		log.Fatal(err)
	}
}
