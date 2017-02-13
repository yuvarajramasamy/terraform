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

	//@TODO convert to test table
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

func TestResourceTimeout_MetaEncode_basic(t *testing.T) {
	// dr := &Resource{
	// 	Timeouts: &ResourceTimeout{
	// 		Create: DefaultTimeout(10 * time.Minute),
	// 		Update: DefaultTimeout(5 * time.Minute),
	// 	},
	// }
	// rt := &ResourceTimeout{
	// 	Create: DefaultTimeout(10 * time.Minute),
	// 	Update: DefaultTimeout(5 * time.Minute),
	// }

	// rt2 := &ResourceTimeout{
	// 	Create:  DefaultTimeout(10 * time.Minute),
	// 	Default: DefaultTimeout(7 * time.Minute),
	// }

	e1 := map[string]interface{}{
		"create": int64(600000000000),
		"update": int64(300000000000),
	}

	e2 := map[string]interface{}{
		"create":  int64(600000000000),
		"update":  int64(420000000000),
		"read":    int64(420000000000),
		"delete":  int64(420000000000),
		"default": int64(420000000000),
	}

	expected := map[string]interface{}{
		TimeoutKey: e1,
	}
	expected2 := map[string]interface{}{
		TimeoutKey: e2,
	}

	cases := []struct {
		Timeout   *ResourceTimeout
		State     *terraform.InstanceDiff
		Expected  map[string]interface{}
		ShouldErr bool
	}{
		// Two fields
		{
			Timeout:   timeoutForValues(10, 0, 5, 0, 0),
			State:     &terraform.InstanceDiff{},
			Expected:  expected,
			ShouldErr: false,
		},
		// Two fields, one is Default
		{
			Timeout:   timeoutForValues(10, 0, 0, 0, 7),
			State:     &terraform.InstanceDiff{},
			Expected:  expected2,
			ShouldErr: false,
		},
		// No fields
		{
			Timeout:   &ResourceTimeout{},
			State:     &terraform.InstanceDiff{},
			Expected:  nil,
			ShouldErr: false,
		},
	}

	for _, c := range cases {
		err := c.Timeout.MetaEncode(c.State)
		log.Printf("\n@@@\npost case meta thing: %s\n@@@\n", spew.Sdump(c.State))
		if err != nil && !c.ShouldErr {
			t.Fatalf("Error, expected:\n%#v\n got:\n%#v\n", c.Expected, c.State.Meta)
		}

		// should maybe just compare [TimeoutKey] but for now we're assuming only
		// that in Meta
		if !reflect.DeepEqual(c.State.Meta, c.Expected) {
			t.Fatalf("Encode not equal, expected:\n%#v\n\ngot:\n%#v\n", c.Expected, c.State.Meta)
		}
	}
}

func timeoutForValues(create, read, update, del, def int) *ResourceTimeout {
	rt := ResourceTimeout{}

	if create != 0 {
		rt.Create = DefaultTimeout(time.Duration(create) * time.Minute)
	}
	if read != 0 {
		rt.Read = DefaultTimeout(time.Duration(read) * time.Minute)
	}
	if update != 0 {
		rt.Update = DefaultTimeout(time.Duration(update) * time.Minute)
	}
	if del != 0 {
		rt.Delete = DefaultTimeout(time.Duration(del) * time.Minute)
	}

	if def != 0 {
		rt.Default = DefaultTimeout(time.Duration(def) * time.Minute)
	}

	return &rt
}
