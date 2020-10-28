package generic

import (
	"log"

	"github.com/PuerkitoBio/goquery"
)

// Navaids descvribes the navigation means available on the airport/
type Navaid struct {
	id              string
	name            string
	frequency       string
	navaidType      string
	magVar          string
	operationsHours string
	position        GeoPosition
	elevation       string
	remarks         string
	key             string
}

func (n *Navaid) SetFromHtmlSelection(tr *goquery.Selection) {
	tr.Find("td").Each(func(index int, td *goquery.Selection) {
		switch index {
		case 0:
			n.setColumn0(td)
		case 1:
			n.id = td.Text()
		case 2:
			n.frequency = td.Text()
		case 3:
			n.operationsHours = td.Text()
		case 4:
			//n.position = td.Text()
			n.setColumn4(td)
		case 5:
			n.elevation = td.Text()
		case 6:
			n.remarks = td.Text()
		}
		n.key = n.id + " " + n.navaidType
	})
}

func (n *Navaid) setColumn0(html *goquery.Selection) {
	var data []string
	fs := html.Text()
	html.Find("p").Each(func(index int, shtml *goquery.Selection) {
		data = append(data, shtml.Text())
	})

	switch len(data) {
	case 1:
		n.navaidType = data[0]
		n.name = fs[0 : len(fs)-len(data[0])]
	case 2:
		n.navaidType = data[0]
		n.magVar = data[1]
		n.name = fs[0 : len(fs)-len(data[0])-len(data[1])]
	}
}

func (n *Navaid) setColumn4(html *goquery.Selection) {
	var data []string
	html.Find("p").Each(func(index int, shtml *goquery.Selection) {
		data = append(data, shtml.Text())
	})

	if len(data) == 2 {
		lat, err := convertDDMMSSSSLatitudeToFloat(data[0])
		if err != nil {
			log.Printf("%s Latitude Conversion problem %s \n", n.name, data[0])
			log.Println(err)
		} else {
			n.position.Latitude = lat
		}

		long, err := convertDDDMMSSSSLongitudeToFloat(data[1])
		if err != nil {
			log.Printf("%s Longitude Conversion problem %s \n", n.name, data[1])
			log.Println(err)
		} else {
			n.position.Longitude = long
		}
	} else {
		log.Printf("%s Conversion problem %s \n", n.name)
	}
}

func (n *Navaid) CompareTo(ext *Navaid) bool {
	if n.key == ext.key {
		return true
	} else {
		if (n.id == ext.id) && (n.navaidType == ext.navaidType) {
			return true
		} else {
			return false
		}
	}
}

func (n *Navaid) IsInMap(m *map[string]Navaid) bool {
	for _, in := range *m {
		if n.CompareTo(&in) {
			return true
		}
	}
	return false
}
