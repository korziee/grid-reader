> [!IMPORTANT]  
> Only works with PNG screenshots of NYT grids (currently).

# Todo

- document the pre processing command
- test the Grid.String() method
- test the /read-grid endpoint
- get a "best attempt" OCR flow going again
- support Sudoku.com grids
- make it work for other file formats

# summary

- `api.go` -> basic web-server to expose grid processing
- `grid.go` -> identifies the grid boundaries, splits out each cell into it's on entity, orchestrates cell processing via `grid_worker.go`
- `grid_worker.go` -> thread pool of cell processors, is orchestrated by the grid, calls processing methods on each cell
- `cell.go` -> in-charge of placeholder and value identification, manages pre-processing via `grid_image.go`
- `grid_image.go` -> low-level wrapper around `image.Image`, executes image pre-processing via ImageMagick, executes OCR via Tesseract

# starting

## docker

- `docker build . -t grid-reader`
- `docker run -p 8080:8080 grid-reader`
- `curl --form file='@grids/3/grid.png' localhost:8080/read-grid`

## local

- you'll need imagemagick installed (https://github.com/gographics/imagick)
- `go run main.go`
