package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		respondErr(err, w)
		return
	}
	log.Printf("parse took %v\n", time.Since(now).Seconds())

	file, header, err := r.FormFile("wallpaper")
	if err != nil {
		respondErr(err, w)
		return
	}
	defer file.Close()

	now = time.Now()
	h := md5.New()
	tee := io.TeeReader(file, h)
	ct := header.Header.Get("Content-Type")
	img, err := convert(tee, ct)
	if err != nil {
		respondErr(err, w)
		return
	}

	sha := fmt.Sprintf("%x", h.Sum(nil))
	log.Printf("h took %v\n", time.Since(now).Seconds())

	now = time.Now()
	filename := sha + ".png"
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, err := os.Create(filename)
		if err != nil {
			respondErr(err, w)
			return
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			respondErr(err, w)
			return
		}
	}
	log.Printf("png encode took %v\n", time.Since(now).Seconds())

	newURL, err := url.Parse(s.webURL)
	if err != nil {
		respondErr(err, w)
		return
	}
	newURL.Path = "/mac-login-wp/download/" + filename

	var t = struct {
		PNGURL string
	}{
		PNGURL: newURL.String(),
	}

	if err := s.tmpl.ExecuteTemplate(w, "upload-body.html", &t); err != nil {
		respondErr(err, w)
	}

}

func download(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/mac-login-wp/download/"):]
	f, err := os.Open(filename)
	if err != nil {
		respondErr(err, w)
		return
	}
	defer f.Close()

	http.ServeContent(w, r, filename, time.Now(), f)
}

func convert(file io.Reader, contentType string) (image.Image, error) {
	var decode func(io.Reader) (image.Image, error)
	switch contentType {
	case "image/jpeg":
		decode = jpeg.Decode
	case "image/png":
		decode = png.Decode
	default:
		return nil, fmt.Errorf("unrecognized image format: %s", contentType)
	}
	src, err := decode(file)
	if err != nil {
		return nil, err
	}

	img := &notOpaqueRGBA{image.NewRGBA(src.Bounds())}
	draw.Draw(img, img.Bounds(), src, image.ZP, draw.Src)

	return img, nil
}

// enforce image.RGBA to always add the alpha channel when encoding PNGs.
type notOpaqueRGBA struct {
	*image.RGBA
}

func (i *notOpaqueRGBA) Opaque() bool {
	return false
}

func respondErr(err error, w http.ResponseWriter) {
	log.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}

type Server struct {
	tmpl   *template.Template
	webURL string
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if err := s.tmpl.ExecuteTemplate(w, "index-body.html", nil); err != nil {
		respondErr(err, w)
		return
	}
}

func main() {
	var (
		flURL = flag.String("http.url", "https://groob.io", "external web url")
	)
	flag.Parse()
	tmpl, err := template.ParseFiles(
		"templates/header.html",
		"templates/index-body.html",
		"templates/upload-body.html",
		"templates/footer.html",
	)
	if err != nil {
		log.Fatal(err)
	}
	s := &Server{
		tmpl:   tmpl,
		webURL: *flURL,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/mac-login-wp/upload", s.handleUpload)
	mux.HandleFunc("/mac-login-wp/download/", download)
	mux.HandleFunc("/mac-login-wp", s.index)
	mux.HandleFunc("/", s.index)
	mux.Handle("/mac-login-wp/css/", http.StripPrefix("/mac-login-wp/css/", http.FileServer(http.Dir("css"))))
	log.Fatal(http.ListenAndServe(":8080", mux))
}
