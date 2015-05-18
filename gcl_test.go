package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func testcmd(cmd string) (string, error) {
	switch {
	case cmd == "sudo ls -a1F /mnt/sda1/var/lib/docker/vfs/dir":
		return currenttest.vs.ls(), nil
	case cmd == "docker ps -aq --no-trunc":
		return "", nil
	case strings.HasPrefix(cmd, "docker inspect -f '{{ .Name }},{{ range $key, $value := .Volumes }}{{ $key }},{{ $value }}##~#{{ end }}' "):
		return "", nil
	case strings.HasPrefix(cmd, "sudo rm /mnt/sda1/var/lib/docker/vfs/dir/"):
		deleted := cmd[len("sudo rm /mnt/sda1/var/lib/docker/vfs/dir/"):]
		deletions = append(deletions, deleted)
		return "", nil
	default:
		return fmt.Sprintf("test '%s'", cmd), errors.New("unknown command")
	}
}

type volspecs []string
type Test struct {
	title string
	vs    volspecs
	res   []int
}

func (vs volspecs) ls() string {
	if len(vs) == 0 {
		return ""
	}
	res := ""
	for i, spec := range vs {
		if strings.HasSuffix(spec, "/") {
			spec = spec[:len(spec)-1]
			res = res + spec + strings.Repeat(fmt.Sprintf("%d", i), 64-len(spec)) + "/\n"
		}
		if strings.HasSuffix(spec, "@") {
			mp := "." + strings.Replace(spec, ";", "###", -1)
			mp = strings.Replace(mp, "/", ",#,", -1)
			res = res + mp + "\n"
		}
	}
	return res
}

var deletions = []string{}
var tests = []Test{
	Test{"empty vfs", []string{}, []int{0, 0, 0, 0, 0}},
	Test{"two volumes", []string{"fa/", "fb/"}, []int{0, 0, 2, 2, 0}},
	Test{"Invalid (ill-formed) markers must be deleted", []string{"cainv/path/a@"}, []int{0, 0, 0, 0, -1}},
	Test{"Invalid (no readlink) markers must be deleted", []string{"ca;/path/a@", "cb;/path/b@"}, []int{0, 0, 0, 0, -2}},
}
var currenttest Test

// TestContainers test different vfs scenarios
func TestContainers(t *testing.T) {
	cmd = testcmd
	for i, test := range tests {
		currenttest = test
		deletions = []string{}
		main()
		tc := Containers()
		toc := OrphanedContainers()
		tv := Volumes()
		tov := OrphanedVolumes()
		tm := Markers()
		if len(tc) != test.res[0] {
			t.Errorf("Test %d: '%s' expected '%d' containers, got '%d'", i+1, test.title, test.res[0], len(tc))
		}
		if len(toc) != test.res[1] {
			t.Errorf("Test %d: '%s' expected '%d' orphaned containers, got '%d'", i+1, test.title, test.res[1], len(toc))
		}
		if len(tv) != test.res[2] {
			t.Errorf("Test %d: '%s' expected '%d' volumes, got '%d'", i+1, test.title, test.res[2], len(tv))
		}
		if len(tov) != test.res[3] {
			t.Errorf("Test %d: '%s' expected '%d' orphaned volumes, got '%d'", i+1, test.title, test.res[3], len(tov))
		}
		if nbmarkers(tm) != test.res[4] {
			t.Errorf("Test %d: '%s' expected '%d' markers, got '%d'", i+1, test.title, test.res[4], nbmarkers(tm))
		}
		fmt.Println("----------------")
	}
}

func nbmarkers(tm markers) int {
	res := len(tm)
	for _, d := range deletions {
		if strings.HasPrefix(d, ".") {
			res = res - 1
		}
	}
	return res
}