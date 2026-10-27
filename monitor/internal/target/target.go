// Package target mendefinisikan service yang dibandingkan (go / rust).
package target

type Target struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
