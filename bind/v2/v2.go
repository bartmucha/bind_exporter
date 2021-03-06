package v2

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/bartmucha/bind_exporter/bind"
)

type Isc struct {
	Bind    Bind     `xml:"bind"`
	XMLName xml.Name `xml:"isc"`
}

type Bind struct {
	Statistics Statistics `xml:"statistics"`
}

type Statistics struct {
	Memory    struct{}         `xml:"memory"`
	Server    Server           `xml:"server"`
	Socketmgr struct{}         `xml:"socketmgr"`
	Taskmgr   bind.TaskManager `xml:"taskmgr"`
	Views     []View           `xml:"views>view"`
}

type Server struct {
	BootTime    time.Time `xml:"boot-time"`
	ConfigTime  time.Time `xml:"config-time"`
	NSStats     []Counter `xml:"nsstat"`
	QueriesIn   []Counter `xml:"queries-in>rdtype"`
	Requests    []Counter `xml:"requests>opcode"`
	SocketStats []Counter `xml:"socketstat"`
	ZoneStats   []Counter `xml:"zonestat"`
}

type View struct {
	Name    string       `xml:"name"`
	Cache   []bind.Gauge `xml:"cache>rrset"`
	Rdtype  []Counter    `xml:"rdtype"`
	Resstat []Counter    `xml:"resstat"`
	Zones   []Counter    `xml:"zones>zone"`
}

type Zone struct {
	Name       string `xml:"name"`
	Rdataclass string `xml:"rdataclass"`
	Serial     string `xml:"serial"`
}

type Counter struct {
	Name    string `xml:"name"`
	Counter uint   `xml:"counter"`
}

// Client implements bind.Client and can be used to query a BIND v2 API.
type Client struct {
	*bind.XMLClient
}

// NewClient returns an initialized Client.
func NewClient(url string, c *http.Client) *Client {
	return &Client{XMLClient: bind.NewXMLClient(url, c)}
}

// Stats implements bind.Stats. The BIND v2 API doesn't provide individual
// endpoints for different statistic groups, the passed parameters don't have
// any effect.
func (c *Client) Stats(...bind.StatisticGroup) (bind.Statistics, error) {
	s := bind.Statistics{}

	root := Isc{}
	if err := c.Get("/", &root); err != nil {
		return s, err
	}
	stats := root.Bind.Statistics

	s.Server.BootTime = stats.Server.BootTime
	for _, t := range stats.Server.QueriesIn {
		s.Server.IncomingQueries = append(s.Server.IncomingQueries, counter(t))
	}
	for _, t := range stats.Server.Requests {
		s.Server.IncomingRequests = append(s.Server.IncomingRequests, counter(t))
	}
	for _, t := range stats.Server.NSStats {
		s.Server.NameServerStats = append(s.Server.NameServerStats, counter(t))
	}
	for _, view := range stats.Views {
		v := bind.View{
			Name:  view.Name,
			Cache: view.Cache,
		}
		for _, t := range view.Rdtype {
			v.ResolverQueries = append(v.ResolverQueries, counter(t))
		}
		for _, t := range view.Resstat {
			v.ResolverStats = append(v.ResolverStats, counter(t))
		}
		s.Views = append(s.Views, v)
	}
	s.TaskManager = stats.Taskmgr

	return s, nil
}

func counter(c Counter) bind.Counter {
	return bind.Counter{
		Name:    c.Name,
		Counter: c.Counter,
	}
}
