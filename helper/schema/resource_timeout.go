package schema

import (
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/copystructure"
)

func DefaultTimeout(tx time.Duration) *time.Duration {
	return &tx
}

type ResourceTimeout struct {
	Create, Read, Update, Delete, Default *time.Duration
}

// ConfigDecode takes a schema and the configuration (available in Diff) and
// validates, parses the timeouts into `t`
func (t *ResourceTimeout) ConfigDecode(s *Resource, c *terraform.ResourceConfig) error {

	log.Printf("\n@@@\nResource timeouts: %s", spew.Sdump(s.Timeouts))
	log.Printf("\n@@@\nConfig timeouts: %s", spew.Sdump(c.Config["timeout"]))

	if s.Timeouts != nil {
		raw, cerr := copystructure.Copy(s.Timeouts)
		if cerr != nil {
			log.Printf("\n@@@\nError with deep copy: %s\n@@@\n", cerr)
		}
		// type assertion
		// rnt := raw.(ResourceTimeout)
		// t = &rnt
		*t = *raw.(*ResourceTimeout)
	}

	log.Printf("what is T now: %s", spew.Sdump(t))

	if v, ok := c.Config["timeout"]; ok {
		raw := v.([]map[string]interface{})
		// raw is []map[string]interface{}
		for _, tv := range raw {
			log.Printf("\n***\n rawT %s", spew.Sdump(tv))
			// rawT := tv.(map[string]interface{})
			for mk, mv := range tv {
				log.Printf("\n$$$$ inner kv: %s // %s", mk, mv.(string))
				keys := []string{"create", "read", "update", "delete", "default"}
				var found bool
				for _, key := range keys {
					if mk == key {
						found = true
						break
					}
				}

				if found {
					log.Printf("\n*** found %s", mk)
				}

				if !found {
					return fmt.Errorf("Unsupported timeout key found (%s)", mk)
				}

				log.Printf("\n***MK: %s\n***t: %#v", mk, t)

				if t.Delete == nil {
					log.Printf("\n***\nt delete is nil\n***\n")
				} else {
					log.Printf("\n***\nt delete is not nil\n***\n")
				}

				if mk == "create" {
					if t.Create == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						log.Printf("\n***\nOverwrote (%s)", mk)
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Create = &rt
						continue
					}
				}

				if mk == "read" {
					if t.Read == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						log.Printf("\n***\nOverwrote (%s)", mk)
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Read = &rt
						continue
					}
				}

				if mk == "update" {
					if t.Update == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						log.Printf("\n***\nOverwrote (%s)", mk)
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Update = &rt
						continue
					}
				}

				if mk == "delete" {
					if t.Delete == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						log.Printf("\n***\nOverwrote (%s)", mk)
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Delete = &rt
						continue
					}
				}

				if mk == "default" {
					if t.Default == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						log.Printf("\n***\nOverwrote (%s)", mk)
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Default = &rt
						continue
					}
				}
			}
		}
	}

	return nil
}

// MetaEncode and MetaDecode are analogous to the Go stdlib JSONEncoder
// interface: they encode/decode a timeouts struct from an instance diff, which is
// where the timeout data is stored after a diff to pass into Apply.
//
// MetaEncode called in Step #2
// MetaDecode called in Step #4
func (t *ResourceTimeout) MetaEncode(*terraform.InstanceDiff) error { return nil }
func (t *ResourceTimeout) MetaDecode(*terraform.InstanceDiff) error { return nil }
