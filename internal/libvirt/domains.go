package libvirt

import (
  "encoding/xml"
  "fmt"
  "io"
  "os"
  "path/filepath"
  "strings"
)

type Domain struct {
  Name string
  UUID string
  MACs []string
}

type domainXML struct {
  XMLName xml.Name `xml:"domain"`
  Name    string   `xml:"name"`
  UUID    string   `xml:"uuid"`
  Devices struct {
    Interfaces []struct {
      MAC struct {
        Address string `xml:"address,attr"`
      } `xml:"mac"`
    } `xml:"interface"`
  } `xml:"devices"`
}

func ScanDomains(qemuDir string) ([]Domain, error) {
  files, err := filepath.Glob(filepath.Join(qemuDir, "*.xml"))
  if err != nil {
    return nil, err
  }

  out := make([]Domain, 0, len(files))
  for _, f := range files {
    d, err := parseDomainFile(f)
    if err != nil {
      // Continue on errors for robustness.
      continue
    }
    if d.Name != "" && d.UUID != "" {
      out = append(out, d)
    }
  }
  return out, nil
}

func parseDomainFile(path string) (Domain, error) {
  fh, err := os.Open(path)
  if err != nil {
    return Domain{}, err
  }
  defer fh.Close()

  b, err := io.ReadAll(fh)
  if err != nil {
    return Domain{}, err
  }

  var dx domainXML
  if err := xml.Unmarshal(b, &dx); err != nil {
    return Domain{}, err
  }

  macs := make([]string, 0)
  for _, iface := range dx.Devices.Interfaces {
    addr := strings.TrimSpace(strings.ToLower(iface.MAC.Address))
    if addr == "" {
      continue
    }
    macs = append(macs, addr)
  }

  return Domain{Name: dx.Name, UUID: dx.UUID, MACs: macs}, nil
}

func (d Domain) String() string {
  return fmt.Sprintf("%s (%s)", d.Name, d.UUID)
}
