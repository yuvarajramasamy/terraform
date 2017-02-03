package schema

import (
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceTimeout_ConfigDecode_badkey(t *testing.T) {
	r := &Resource{
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(10 * time.Minute),
			Update: DefaultTimeout(5 * time.Minute),
		},
	}

	raw, err := config.NewRawConfig(
		map[string]interface{}{
			"foo": "bar",
			"timeout": []map[string]interface{}{
				map[string]interface{}{
					"create": "2m",
				},
				map[string]interface{}{
					"delete": "1m",
				},
			},
		})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	c := terraform.NewResourceConfig(raw)

	timeout := &ResourceTimeout{}
	err = timeout.ConfigDecode(r, c)
	if err == nil {
		log.Println("Expected bad timeout key")
		t.Fatalf("err: %s", err)
	}

	log.Printf("\n***\nWhat is timeout: %s", spew.Sdump(timeout))
}

func TestResourceTimeout_ConfigDecode(t *testing.T) {
	r := &Resource{
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(10 * time.Minute),
			Update: DefaultTimeout(5 * time.Minute),
		},
	}

	raw, err := config.NewRawConfig(
		map[string]interface{}{
			"foo": "bar",
			"timeout": []map[string]interface{}{
				map[string]interface{}{
					"create": "2m",
				},
				map[string]interface{}{
					"update": "1m",
				},
			},
		})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	c := terraform.NewResourceConfig(raw)

	timeout := &ResourceTimeout{}
	err = timeout.ConfigDecode(r, c)
	if err != nil {
		log.Println("Expected good timeout returned")
		t.Fatalf("err: %s", err)
	}

	expected := &ResourceTimeout{
		Create: DefaultTimeout(2 * time.Minute),
		Update: DefaultTimeout(1 * time.Minute),
	}

	log.Printf("\n***\nWhat is timeout: %s", spew.Sdump(timeout))
	if !reflect.DeepEqual(timeout, expected) {
		t.Fatalf("bad timeout decode, expected (%#v), got (%#v)", expected, timeout)
	}
}
