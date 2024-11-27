#!/bin/bash


INPUT_DIR="./samples"
OUTPUT_DIR="./sample-outputs"

rm -rf $OUTPUT_DIR

mkdir -p "$OUTPUT_DIR"

for image in "$INPUT_DIR"/*.png; do
    filename=$(basename "$image")
    sample_output_dir="$OUTPUT_DIR/$filename"
    mkdir -p $sample_output_dir

    tesseract "$image" "$sample_output_dir/10" --psm 10 tsv hocr 2>> "$sample_output_dir/10.txt"
    sed -i '' '/<\/body>/i\
      <script src="https://unpkg.com/hocrjs"></script>
      ' "$sample_output_dir/10.hocr"

    tesseract "$image" "$sample_output_dir/6" --psm 6 -c tessedit_char_whitelist=123456789 tsv hocr 2>> "$sample_output_dir/6.txt"
    sed -i '' '/<\/body>/i\
      <script src="https://unpkg.com/hocrjs"></script>
      ' "$sample_output_dir/6.hocr"

done

find $OUTPUT_DIR -name "*.hocr" -print0 | xargs -0 -I {} mv "{}" {}.html


echo "Processing complete. Output files saved to $OUTPUT_DIR"
