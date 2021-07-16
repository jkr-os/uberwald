package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"html"
	"net/http"
	"strings"
	"text/template"
	"time"

	"strconv"

	"golang.org/x/net/context"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

// A GeometryType serves to enumerate the different GeoJSON geometry types.
type GeometryType string

// A Geometry correlates to a GeoJSON geometry object.
type Geometry struct {
	Type            GeometryType `json:"type,omitempty"`
	BoundingBox     []float64    `json:"bbox,omitempty"`
	Point           []float64
	MultiPoint      [][]float64
	LineString      [][]float64
	MultiLineString [][][]float64
	Polygon         [][][]float64
	MultiPolygon    [][][][]float64
	Geometries      []*Geometry
	CRS             map[string]interface{} `json:"crs,omitempty"` // Coordinate Reference System Objects are not currently supported
}

// A Properties correlates to a GeoJSON properties object.
type Properties struct {
	FID       uint64 `json:"fid,omitempty"`
	Gebiet    string `json:"gebiet,omitempty"`
	PatenID   int    `json:"patenid,omitempty"`
	Patenfeld string `json:"patenfeld,omitempty"`
	RasterID  uint64 `json:"rasterid,omitempty"`
}

// A Feature corresponds to GeoJSON feature object
type Feature struct {
	ID          interface{}            `json:"id,omitempty"`
	Type        string                 `json:"type,omitempty"`
	BoundingBox []float64              `json:"bbox,omitempty"`
	Geometry    *Geometry              `json:"geometry,omitempty"`
	Properties  *Properties            `json:"properties,omitempty"`
	CRS         map[string]interface{} `json:"crs,omitempty"` // Coordinate Reference System Objects are not currently supported
}

func sendJS(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "ui/js/geojsonhint.js")
}

func update(w http.ResponseWriter, r *http.Request) {

	// Use the template.ParseFiles() function to read the template file into a
	// template set. If there's an error, we log the detailed error message and use
	// the http.Error() function to send a generic 500 Internal Server Error
	// response to the user.
	ts, err := template.ParseFiles("./ui/html/update.page.tmpl")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// We then use the Execute() method on the template set to write the template
	// content as the response body. The last parameter to Execute() represents any
	// dynamic data that we want to pass in, which for now we'll leave as nil.
	err = ts.Execute(w, nil)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
	}
}

func upload(w http.ResponseWriter, r *http.Request) {

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("myFile")
	projectname, err := r.FormValue("projectname")
	if err != nil {
		log.Println("Error Retrieving the File")
		log.Println(err)
		fmt.Fprintf(w, " Error Retrieving the File %v...", err)
		return
	}
	defer file.Close()
	log.Printf("Uploaded File: %+v\n", handler.Filename)
	log.Printf("File Size: %+v\n", handler.Size)
	log.Printf("MIME Header: %+v\n", handler.Header)
	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, " Error Reading the File %v...", err)
	}

	ctx := context.Background()
	opt := option.WithCredentialsFile(credentialsFile)
	config := &firebase.Config{DatabaseURL: databaseURL}
	app, err := firebase.NewApp(ctx, config, opt)

	if err != nil {
		panic(fmt.Sprintf("error initializing app: %v", err))
	}

	client, err := app.Database(ctx)

	if err != nil {
		log.Fatalln("Error initializing database client:", err)
	}

	ref,err := client.NewRef("wildnispate")

	if err != nil {
		log.Fatalln("Error opening database:", err)
	}

	child, err := ref.Child(projectname)
	if err != nil {
		ref.Update(ctx, map[string]interface{}{ projectname, fileBytes }
	} else {
		ref.Set(ctx, map[string]interface{}{ projectname, fileBytes }
	}
	ts, err := template.ParseFiles("./ui/html/received.page.tmpl")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// We then use the Execute() method on the template set to write the template
	// content as the response body. The last parameter to Execute() represents any
	// dynamic data that we want to pass in, which for now we'll leave as nil.
	err = ts.Execute(w, nil)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
	}

}

func hektar(w http.ResponseWriter, r *http.Request) {

	//id, err := strconv.Atoi(r.URL.Query().Get("id"))

	queryIDs := strings.Split(r.URL.Query().Get("id"), ",")
	area, err := r.URL.Query().Get("area")
	areaurl := "biesenthalerbecken/features"
	if err == nil {
		areaurl = "wildnispate/" + html.UnescapeString(area) + "/features"
	}

	ctx := context.Background()
	opt := option.WithCredentialsFile(credentialsFile)
	config := &firebase.Config{DatabaseURL: databaseURL}
	app, err := firebase.NewApp(ctx, config, opt)

	if err != nil {
		panic(fmt.Sprintf("error initializing app: %v", err))
	}

	client, err := app.Database(ctx)

	if err != nil {
		log.Fatalln("Error initializing database client:", err)
	}

	ref := client.NewRef(areaurl)
	var features []Feature

	if err := ref.Get(ctx, &features); err != nil {
		log.Fatalln("Error reading from database:", err)
	}

	var found int

	for _, queryID := range queryIDs {

		id, _ := strconv.ParseUint(queryID, 10, 64)

		for index, element := range features {

			if element.Properties.RasterID == id {

				// if found
				found++

				if err := ref.Child(strconv.Itoa(index)).Update(ctx, map[string]interface{}{
					"properties/PatenID": 1,
				}); err != nil {
					log.Fatalln("Error updating child:", err)
				}

				fmt.Fprintf(w, "Put hektar with ID %v...", element.Properties.RasterID)

				break

			}
		}

	}

	if found == 0 {
		http.NotFound(w, r)
		return
	}

}
