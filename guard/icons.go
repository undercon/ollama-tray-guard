package guard

// generateIcon creates a minimal valid ICO file (16x16, 32-bit BGRA) with a filled circle.
// Windows systray (getlantern/systray) requires ICO format on Windows.

import (
	"encoding/binary"
	"image/color"
	"math"
)

func generateIcon(c color.RGBA) []byte {
	const size = 16

	// Build pixel data (BGRA, bottom-up row order)
	pixels := make([]byte, size*size*4)
	cx, cy := 7.5, 7.5
	radius := 6.5
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			// bottom-up: flip row
			row := size - 1 - y
			idx := (row*size + x) * 4
			if math.Sqrt(dx*dx+dy*dy) <= radius {
				pixels[idx+0] = c.B
				pixels[idx+1] = c.G
				pixels[idx+2] = c.R
				pixels[idx+3] = c.A
			}
		}
	}

	// AND mask: all zeros = opaque (size rows, each row padded to 4-byte boundary)
	// For 16px wide: 16 bits = 2 bytes/row, padded to 4 bytes
	andMask := make([]byte, size*4)

	// BITMAPINFOHEADER (40 bytes)
	dibHeader := make([]byte, 40)
	binary.LittleEndian.PutUint32(dibHeader[0:], 40)        // biSize
	binary.LittleEndian.PutUint32(dibHeader[4:], size)      // biWidth
	binary.LittleEndian.PutUint32(dibHeader[8:], size*2)    // biHeight (doubled for ICO)
	binary.LittleEndian.PutUint16(dibHeader[12:], 1)        // biPlanes
	binary.LittleEndian.PutUint16(dibHeader[14:], 32)       // biBitCount
	// rest zeros: BI_RGB compression, etc.

	imageDataSize := uint32(len(dibHeader) + len(pixels) + len(andMask))

	// ICO header (6 bytes) + ICONDIRENTRY (16 bytes) = 22 bytes offset to image data
	ico := make([]byte, 6+16+int(imageDataSize))

	// ICONDIR
	binary.LittleEndian.PutUint16(ico[0:], 0)    // reserved
	binary.LittleEndian.PutUint16(ico[2:], 1)    // type = ICO
	binary.LittleEndian.PutUint16(ico[4:], 1)    // count = 1

	// ICONDIRENTRY
	ico[6] = size  // width
	ico[7] = size  // height
	ico[8] = 0     // colorCount (0 = >256 colors)
	ico[9] = 0     // reserved
	binary.LittleEndian.PutUint16(ico[10:], 1)           // planes
	binary.LittleEndian.PutUint16(ico[12:], 32)          // bitCount
	binary.LittleEndian.PutUint32(ico[14:], imageDataSize) // size of image data
	binary.LittleEndian.PutUint32(ico[18:], 22)          // offset to image data

	// Image data
	off := 22
	copy(ico[off:], dibHeader)
	off += len(dibHeader)
	copy(ico[off:], pixels)
	off += len(pixels)
	copy(ico[off:], andMask)

	return ico
}

var (
	IconGreen  = generateIcon(color.RGBA{R: 76, G: 175, B: 80, A: 255})
	IconYellow = generateIcon(color.RGBA{R: 255, G: 193, B: 7, A: 255})
	IconRed    = generateIcon(color.RGBA{R: 244, G: 67, B: 54, A: 255})
)
