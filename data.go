package gosiklu

type Attribute struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type Info struct {
	Type         string      `xml:"type,attr"`
	Details      string      `xml:"details,attr"`
	Name         string      `xml:"name,attr"`
	Attr         []Attribute `xml:"attr"`
	StatsHeader  string      `xml:"stats-header"`
	StatsCurrent string      `xml:"stats-current"`
}

type SikluData struct {
	Request string `xml:"request"`
	Mo      []Info `xml:"mo"`
}

func (s *SikluData) GetAttrValue(info, attribute string) string {
	x, _ := s.GetAttr(info, attribute)
	return x
}

func (s *SikluData) GetAttr(info, attribute string) (string, bool) {
	for _, mo := range s.Mo {
		if mo.Name == info {
			for _, attr := range mo.Attr {
				if attr.Name == attribute {
					return attr.Value, true
				}
			}
		}
	}
	return "", false
}

func (s *SikluData) GetInfoByName(name string) *Info {
	for _, mo := range s.Mo {
		if mo.Name == name {
			info := new(Info)
			*info = mo
			return info
		}
	}
	return nil
}

func (s *SikluData) GetInfoByType(typeName string) []Info {
	var infos []Info
	for _, mo := range s.Mo {
		if mo.Type == typeName {
			infos = append(infos, mo)
		}
	}
	if len(infos) > 0 {
		return infos
	}
	return nil
}

// CompareInfo compares two Info structs and returns true if they are the same
// The comparison is done by comparing the Type, Name, and Attr fields
// The two sets of Attributes must be the same length and have the same Name and Value
// in order to be considered the same.  The order of the Attributes is not important.
func CompareInfo(a, b *Info, ignoreAttrs []string) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Type != b.Type || a.Name != b.Name {
		return false
	}
	if a.Attr == nil || b.Attr == nil {
		if a.Attr == nil && b.Attr == nil {
			return true
		}
		return false
	}
	matches := 0
OUTER:
	for i := range a.Attr {
		if ignoreAttrs != nil {
			for _, ignore := range ignoreAttrs {
				if a.Attr[i].Name == ignore {
					matches++ // pretend it matched
					continue OUTER
				}
			}
		}
		for j := range b.Attr {
			if a.Attr[i].Name == b.Attr[j].Name && a.Attr[i].Value == b.Attr[j].Value {
				matches++
				break
			} else if a.Attr[i].Name == b.Attr[j].Name {
			}
		}
	}
	if matches != len(a.Attr) {
		return false
	}
	return true
}
