> [!IMPORTANT]  
> Only works with PNG screenshots of NYT grids currently.

# Todo

- add support for command line interaction (filename, mode, and debug flags)
- document the pre processing command
- test the Grid.String() method
- test the /grid endpoint
- support Sudoku.com grids
- get a "best attempt" OCR flow going again
- make work for other file formats

# summary

- `api.go` -> basic web-server to expose grid processing
- `grid.go` -> identifies the grid boundaries, splits out each cell into it's on entity, orchestrates cell processing via `grid_worker.go`
- `grid_worker.go` -> thread pool of cell processors, is orchestrated by the grid, calls processing methods on each cell
- `cell.go` -> in-charge of placeholder and value identification, manages pre-processing via `grid_image.go`
- `grid_image.go` -> low-level wrapper around `image.Image`, executes image pre-processing via ImageMagick, executes OCR via Tesseract
