package template

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	moby "github.com/moby/tool/src/moby"
	yaml "gopkg.in/yaml.v2"
)

//Template is the data on which templating functions can be executed on
type Template struct {
	Moby  moby.Moby
	Repos []string
}

//Result is the result of templating
type Result struct {
	Moby moby.Moby
	Subs []Substitution
}

//Substitution contains the template, source of templating information and result of a templating action
type Substitution struct {
	Template string
	Source   string
	Result   string
}

type imageTemplate struct {
	image           *moby.Image
	repo, name, tag string
	//needed for init images
	value *string
}

//Bake searches for placeHolder in configs image label and substitutes them
func (t Template) Bake() (Result, error) {
	m, err := copy(t.Moby)
	result := Result{Subs: []Substitution{}}
	if err != nil {
		return result, err
	}

	if len(t.Repos) < 1 {
		return result, errors.New("no pkg repo")
	}

	imageTemplates := getImageTemplates(&m)
	if len(imageTemplates) == 0 {
		return result, nil
	}

	for _, repo := range t.Repos {
		subs, err := walkRepo(repo, &imageTemplates)

		result.Subs = append(result.Subs, subs...)
		if err != nil {
			return result, err
		}
	}
	var errs []string
	for _, imageString := range imageTemplates {
		errs = append(errs, fmt.Sprintln("could not find pkg for:", *imageString.value))
	}
	if len(errs) != 0 {
		return result, errors.New(strings.Join(errs, ""))
	}

	result.Moby = m
	return result, err
}

//poor mens deep copy
func copy(m moby.Moby) (moby.Moby, error) {
	var r moby.Moby
	b, err := yaml.Marshal(m)
	if err != nil {
		return r, err
	}
	r, err = moby.NewConfig(b)
	if err != nil {
		return r, err
	}
	return r, nil
}

func walkRepo(repo string, imageTemplates *[]imageTemplate) ([]Substitution, error) {
	var subs []Substitution
	repoFile, err := os.Open(repo)
	if err != nil {
		return subs, err
	}

	pkgs, err := repoFile.Readdirnames(-1)
	repoFile.Close()
	if err != nil {
		return subs, err
	}

	sort.Strings(pkgs)
	for _, pkgPath := range pkgs {

		pkg, err := pkglib.New(filepath.Join(repo, pkgPath))
		if err != nil {
			continue
		}

		for i := len(*imageTemplates) - 1; i >= 0; i-- {
			if (*imageTemplates)[i].name == pkg.Image() {
				sub, err := substitute((*imageTemplates)[i], filepath.Join(repo, pkgPath), pkg)
				if err != nil {
					return subs, err
				}
				subs = append(subs, sub)
				*imageTemplates = append((*imageTemplates)[:i], (*imageTemplates)[i+1:]...)
			}
		}
		if len(*imageTemplates) == 0 {
			return subs, nil
		}

	}
	return subs, err
}

func substitute(imageString imageTemplate, path string, p pkglib.Pkg) (Substitution, error) {
	hash := p.Hash()

	if hash == "" {
		return Substitution{}, errors.New("could not retrieve hash for " + *imageString.value)
	}

	old := *imageString.value
	*imageString.value = fmt.Sprintf("%v/%v:%v", imageString.repo, imageString.name, hash)
	return Substitution{Template: old, Source: path, Result: *imageString.value}, nil
}

func getImageTemplates(m *moby.Moby) []imageTemplate {
	var images []*moby.Image
	var tagTemplates []imageTemplate

	images = append(images, m.Onboot...)
	images = append(images, m.Services...)
	images = append(images, m.Onshutdown...)

	addToTagTemplates := func(imageString *string, image *moby.Image) {
		tagSplit := strings.Split(*imageString, ":")
		tag := tagSplit[len(tagSplit)-1]
		if isTagTemplate(tag) {
			imageSplit := strings.Split(tagSplit[0], "/")
			name := imageSplit[1]
			repo := imageSplit[0]
			tagTemplates = append(tagTemplates, imageTemplate{
				tag:   tag,
				name:  name,
				repo:  repo,
				value: imageString,
				image: image})
		}
	}

	for _, v := range images {
		addToTagTemplates(&v.Image, v)
	}
	for i := range m.Init {
		addToTagTemplates(&m.Init[i], nil)
	}

	return tagTemplates
}

func isTagTemplate(tag string) bool {
	return tag[:1] == "<" && tag[len(tag)-1:] == ">"
}
