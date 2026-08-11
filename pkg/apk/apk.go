package apk

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gitlab.alpinelinux.org/alpine/go/repository"
)

type PackagePerAlpineVersionAndArchitecture map[string]map[string]map[string][]*repository.Package

type Context struct {
	client                                  *http.Client
	indexURLs                               []string
	packagesPerAlpineVersionAndArchitecture PackagePerAlpineVersionAndArchitecture
}

func New(client *http.Client, indexURLs []string) Context {
	return Context{
		client:                                  client,
		indexURLs:                               indexURLs,
		packagesPerAlpineVersionAndArchitecture: make(PackagePerAlpineVersionAndArchitecture),
	}
}

func (c Context) AddAlpineVersionAndArchitecture(alpineVersion, architecture string) error {
	if _, ok := c.packagesPerAlpineVersionAndArchitecture[alpineVersion]; !ok {
		c.packagesPerAlpineVersionAndArchitecture[alpineVersion] = make(map[string]map[string][]*repository.Package)
	}
	if _, ok := c.packagesPerAlpineVersionAndArchitecture[alpineVersion][architecture]; !ok {
		c.packagesPerAlpineVersionAndArchitecture[alpineVersion][architecture] = nil
	}
	if c.packagesPerAlpineVersionAndArchitecture[alpineVersion][architecture] == nil || len(c.packagesPerAlpineVersionAndArchitecture[alpineVersion][architecture]) == 0 {
		return c.GetApkPackagesForAlpineVersionAndArchitecture(alpineVersion, architecture)
	}
	return nil
}

func (c Context) GetApkPackages() (*PackagePerAlpineVersionAndArchitecture, []error) {
	var errors []error
	for alpineVersion, packagesPerArchitecture := range c.packagesPerAlpineVersionAndArchitecture {
		for architecture := range packagesPerArchitecture {
			err := c.GetApkPackagesForAlpineVersionAndArchitecture(alpineVersion, architecture)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	return &c.packagesPerAlpineVersionAndArchitecture, errors
}

func (c Context) GetApkPackagesForAlpineVersionAndArchitecture(alpineVersion, architecture string) error {
	var packages []*repository.Package

	for _, url := range c.indexURLs {
		url = strings.Replace(url, "{alpineVersion}", alpineVersion, -1)
		url = strings.Replace(url, "{architecture}", architecture, -1)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return errors.Join(err, fmt.Errorf("failed getting URI %s", url))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non ok http response for URI %s code: %v", url, resp.StatusCode)
		}

		ps, err := parseApkIndex(resp.Body)
		if err != nil {
			return err
		}

		packages = append(packages, ps...)
	}
	c.packagesPerAlpineVersionAndArchitecture[alpineVersion][architecture] = getPackagesMap(packages)

	return nil
}

func parseApkIndex(indexData io.ReadCloser) ([]*repository.Package, error) {
	apkIndex, err := repository.IndexFromArchive(indexData)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("failed to parse response %v", indexData))
	}

	return apkIndex.Packages, nil
}

func getPackagesMap(packages []*repository.Package) map[string][]*repository.Package {
	packageMap := make(map[string][]*repository.Package)
	for _, p := range packages {
		packageMap[p.Name] = append(packageMap[p.Name], p)

		for _, provide := range p.Provides {
			if strings.Contains(provide, ":") {
				continue
			}

			name, _, _ := strings.Cut(provide, "=")
			packageMap[name] = append(packageMap[name], p)
		}
	}
	return packageMap
}
