# streaming cache

single writer multi reader "cache". read while writing.

```go
func main() {

	t := scache.New()

	ht := http.Client{}
	r, _ := ht.Get("http://bigFile.dat")
	go func() {
		io.Copy(t, r.Body)
		r.Body.Close()
		t.Done()
	}()

	// OR File("500mb.blob").StreamTo(t)

	http.HandleFunc("/out.dat", func(w http.ResponseWriter, r *http.Request) {
		//TODO: copy headers
		t.ReplayAndSubscribeTo(w)
	})

	http.ListenAndServe(":9002", nil)

	runtime.Goexit()
}
```
