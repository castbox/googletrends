package gogtrends

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"

	"log"

	"github.com/pkg/errors"
)

const (
	headerKeyAccept      = "Accept"
	headerKeyCookie      = "Cookie"
	headerKeySetCookie   = "Set-Cookie"
	headerKeyContentType = "Content-Type"
	contentTypeJSON      = "application/json"
	contentTypeForm      = "application/x-www-form-urlencoded;charset=UTF-8"
)

type gClient struct {
	c         *http.Client
	defParams url.Values

	tcm        *sync.RWMutex
	trendsCats map[string]string

	cm          *sync.RWMutex
	exploreCats *ExploreCatTree

	lm          *sync.RWMutex
	exploreLocs *ExploreLocTree

	cookie string
	debug  bool
}

func newGClient() *gClient {
	// default request params
	p := make(url.Values)
	for k, v := range defaultParams {
		p.Add(k, v)
	}

	return &gClient{
		c:          http.DefaultClient,
		defParams:  p,
		tcm:        new(sync.RWMutex),
		trendsCats: trendsCategories,
		cm:         new(sync.RWMutex),
		lm:         new(sync.RWMutex),
	}
}

func (c *gClient) defaultParams() url.Values {
	out := make(map[string][]string, len(c.defParams))
	for i, v := range c.defParams {
		out[i] = make([]string, len(v))
		copy(out[i], v)
	}

	return out
}

func (c *gClient) getCategories() *ExploreCatTree {
	c.cm.RLock()
	defer c.cm.RUnlock()
	return c.exploreCats
}

func (c *gClient) setCategories(cats *ExploreCatTree) {
	c.cm.Lock()
	defer c.cm.Unlock()
	c.exploreCats = cats
}

func (c *gClient) getLocations() *ExploreLocTree {
	c.lm.RLock()
	defer c.lm.RUnlock()
	return c.exploreLocs
}

func (c *gClient) setLocations(locs *ExploreLocTree) {
	c.lm.Lock()
	defer c.lm.Unlock()
	c.exploreLocs = locs
}

func (c *gClient) do(ctx context.Context, u *url.URL) ([]byte, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, errCreateRequest)
	}

	r.Header.Add(headerKeyAccept, contentTypeJSON)

	if len(client.cookie) != 0 {
		r.Header.Add(headerKeyCookie, client.cookie)
	}

	if client.debug {
		log.Println("[Debug] Request with params: ", r.URL)
	}

	resp, err := c.c.Do(r)
	if err != nil {
		return nil, errors.Wrap(err, errDoRequest)
	}
	defer resp.Body.Close()

	if client.debug {
		log.Println("[Debug] Response: ", resp)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		cookie := strings.Split(resp.Header.Get(headerKeySetCookie), ";")
		if len(cookie) > 0 {
			client.cookie = cookie[0]
			r.Header.Set(headerKeyCookie, cookie[0])

			resp, err = c.c.Do(r)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(ErrRequestFailed, errReqDataF, resp.StatusCode, resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

// doPost performs a POST request to the specified URL with the given payload
func (c *gClient) doPost(ctx context.Context, u *url.URL, payload string) ([]byte, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(payload))
	if err != nil {
		return nil, errors.Wrap(err, errCreateRequest)
	}

	r.Header.Add(headerKeyContentType, contentTypeForm)

	if len(client.cookie) != 0 {
		r.Header.Add(headerKeyCookie, client.cookie)
	}

	if client.debug {
		log.Println("[Debug] POST Request with params: ", r.URL)
		log.Println("[Debug] POST Request payload: ", payload)
	}

	resp, err := c.c.Do(r)
	if err != nil {
		return nil, errors.Wrap(err, errDoRequest)
	}
	defer resp.Body.Close()

	if client.debug {
		log.Println("[Debug] Response: ", resp)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		cookie := strings.Split(resp.Header.Get(headerKeySetCookie), ";")
		if len(cookie) > 0 {
			client.cookie = cookie[0]
			r.Header.Set(headerKeyCookie, cookie[0])

			resp, err = c.c.Do(r)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(ErrRequestFailed, errReqDataF, resp.StatusCode, resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

func (c *gClient) unmarshal(str string, dest interface{}) error {
	if err := jsoniter.UnmarshalFromString(str, dest); err != nil {
		return errors.Wrap(err, errParsing)
	}

	return nil
}

// extractJSONFromResponse extracts the nested JSON object from the API response
func (c *gClient) extractJSONFromResponse(text string) ([]string, error) {
	if client.debug {
		log.Println("[Debug] Extracting JSON from API response")
	}

	var result []string

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			try := func() error {
				var intermediate []interface{}
				if err := jsoniter.UnmarshalFromString(trimmed, &intermediate); err != nil {
					return err
				}

				if len(intermediate) > 0 && len(intermediate) > 2 {
					secondItem, ok := intermediate[0].([]interface{})
					if !ok || len(secondItem) < 3 {
						return errors.New("invalid intermediate format")
					}

					jsonStr, ok := secondItem[2].(string)
					if !ok {
						return errors.New("invalid json string format")
					}

					var data []interface{}
					if err := jsoniter.UnmarshalFromString(jsonStr, &data); err != nil {
						return err
					}

					if len(data) > 1 {
						items, ok := data[1].([]interface{})
						if !ok {
							return errors.New("invalid items format")
						}

						for _, item := range items {
							if itemArr, ok := item.([]interface{}); ok && len(itemArr) > 0 {
								if topic, ok := itemArr[0].(string); ok {
									result = append(result, topic)
								}
							}
						}
					}
				}
				return nil
			}

			if err := try(); err != nil {
				if client.debug {
					log.Println("[Debug] Error parsing JSON:", err)
				}
				continue
			}

			if len(result) > 0 {
				if client.debug {
					log.Println("[Debug] JSON extraction successful")
				}
				return result, nil
			}
		}
	}

	return nil, errors.New("no valid JSON found in response")
}

func (c *gClient) trends(ctx context.Context, path, hl, loc string, args ...map[string]string) (string, error) {
	u, _ := url.Parse(path)

	// required params
	p := client.defaultParams()
	if len(loc) > 0 {
		p.Set(paramGeo, loc)
	}
	p.Set(paramHl, hl)

	// additional params
	if len(args) > 0 {
		for _, arg := range args {
			for n, v := range arg {
				p.Set(n, v)
			}
		}
	}

	u.RawQuery = p.Encode()

	data, err := client.do(ctx, u)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (c *gClient) validateCategory(cat string) bool {
	c.tcm.RLock()
	_, ok := client.trendsCats[cat]
	c.tcm.RUnlock()

	return ok
}

// trendsNew uses the new Google Trends API to fetch trending searches
func (c *gClient) trendsNew(ctx context.Context, hl, loc string) ([]string, error) {
	u, _ := url.Parse(gBatchExecute)

	// Create payload for the new API
	payload := fmt.Sprintf("f.req=[[[i0OFE,\"[null, null, \\\"%s\\\", 0, null, 48]\"]]]", loc)

	if client.debug {
		log.Println("[Debug] Using new Google Trends API with payload:", payload)
	}

	data, err := client.doPost(ctx, u, payload)
	if err != nil {
		return nil, err
	}

	return client.extractJSONFromResponse(string(data))
}
