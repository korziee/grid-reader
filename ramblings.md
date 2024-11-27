want: a conversion that transforms all cases of an image into the same representation.
cases:

1. empty
2. digit
3. digit (yellow bg)
4. digit (gray bg)
5. placeholders
6. placeholders (yellow bg)

my idea: find the majority colour (i.e. the background if not black) and convert it to alpha

I can use `magick debug/R1C1.png -colorspace Gray -identify -verbose -format "%[fx:255*median]" info:` to find the dom color

I can use `magick debug/R3C4.png -fuzz 0% -fill none -opaque 'rgb(255,255,255)' -alpha extract -threshold 0 -negate -transparent white -trim r3c4.png` to strip the background and turn everything black

I can use `magick compare -colorspace Gray -alpha off -metric AE r1c2.png r3c4.png _` to get a pixel difference

ok. so. the process:

1. convert each cell to grayscale
2. find the dominant colour (the bg)
3. strip that and turn to black
4. compare using AE
5. calculate confidence (100 - (error/pixels \* 100))
6. if > 95% we've probably got a match

Next steps (digits only):

1. split tests out into values vs placeholders
2. create a value dictionary for every digit NYT
   1. grayscaled, thresholded, and trimmed image (i.e. magick debug/R3C4.png -fuzz 0% -fill none -opaque 'rgb(255,255,255)' -alpha extract -threshold 0 -negate -transparent white -trim r3c4.png)
3. when iterating over cells, apply the same transform
4. compare each cell to the dictionary value, if we have confidence of 95% or greater, accept it

Next steps (placeholders):

1. create a value dictionary for every placeholder NYT
   1. will require cropping, use a white bg for dict values
   2. apply transform to each of them
2. when iterating over cells, if no matches for digits, then further extract all nine placeholders by their well known positions
3. for each placeholder run them through the the dictionary values for placeholders

May need to run `export CGO_CFLAGS_ALLOW='-Xpreprocessor'` first before building
