package render

import (
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// SavePNG reads the default framebuffer (w x h) into a PNG file.
func SavePNG(path string, w, h int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return CapturePNG(f, w, h)
}

// CapturePNG writes the default framebuffer with standard PNG compression.
func CapturePNG(dst io.Writer, w, h int) error {
	px := make([]byte, w*h*4)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadPixels(0, 0, int32(w), int32(h), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&px[0]))
	return (&CapturedFrame{width: w, height: h, pixels: px}).WritePNG(dst)
}

type CapturedFrame struct {
	width, height int
	pixels        []byte
}

// WritePNG encodes the captured pixels and can run without the OpenGL context.
func (f *CapturedFrame) WritePNG(dst io.Writer) error {
	img := image.NewNRGBA(image.Rect(0, 0, f.width, f.height))
	stride := img.Stride
	for y := 0; y < f.height; y++ {
		source := (f.height - 1 - y) * stride
		copy(img.Pix[y*stride:(y+1)*stride], f.pixels[source:source+stride])
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	return png.Encode(dst, img)
}

// FramebufferReadback stages one frame in a pixel pack buffer. The render
// thread only copies its bytes after the GPU fence signals.
type FramebufferReadback struct {
	buffer        uint32
	fence         uintptr
	width, height int
	size          int
}

func (r *FramebufferReadback) Pending() bool { return r.fence != 0 }

func (r *FramebufferReadback) Start(w, h int) error {
	if r.Pending() {
		return errors.New("previous screenshot is still being read")
	}
	if w <= 0 || h <= 0 {
		return errors.New("empty screenshot framebuffer")
	}
	if r.buffer == 0 {
		gl.CreateBuffers(1, &r.buffer)
		if r.buffer == 0 {
			return errors.New("unable to create screenshot buffer")
		}
	}
	size := w * h * 4
	if size != r.size {
		gl.NamedBufferData(r.buffer, size, nil, gl.STREAM_READ)
		var allocated int64
		gl.GetNamedBufferParameteri64v(r.buffer, gl.BUFFER_SIZE, &allocated)
		if allocated != int64(size) {
			return errors.New("unable to allocate screenshot buffer")
		}
		removeGPUBytes(uint64(r.size))
		addGPUBytes(uint64(size))
		r.size = size
	}
	r.width, r.height = w, h
	var previous, alignment int32
	gl.GetIntegerv(gl.PIXEL_PACK_BUFFER_BINDING, &previous)
	gl.GetIntegerv(gl.PACK_ALIGNMENT, &alignment)
	gl.BindBuffer(gl.PIXEL_PACK_BUFFER, r.buffer)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadPixels(0, 0, int32(w), int32(h), gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.PixelStorei(gl.PACK_ALIGNMENT, alignment)
	gl.BindBuffer(gl.PIXEL_PACK_BUFFER, uint32(previous))
	r.fence = gl.FenceSync(gl.SYNC_GPU_COMMANDS_COMPLETE, 0)
	if r.fence == 0 {
		return errors.New("unable to start screenshot readback")
	}
	return nil
}

func (r *FramebufferReadback) Poll() (*CapturedFrame, error) {
	if !r.Pending() {
		return nil, nil
	}
	status := gl.ClientWaitSync(r.fence, gl.SYNC_FLUSH_COMMANDS_BIT, 0)
	if status == gl.TIMEOUT_EXPIRED {
		return nil, nil
	}
	gl.DeleteSync(r.fence)
	r.fence = 0
	if status != gl.ALREADY_SIGNALED && status != gl.CONDITION_SATISFIED {
		return nil, errors.New("screenshot readback failed")
	}
	px := make([]byte, r.size)
	gl.GetNamedBufferSubData(r.buffer, 0, len(px), unsafe.Pointer(&px[0]))
	return &CapturedFrame{width: r.width, height: r.height, pixels: px}, nil
}

func (r *FramebufferReadback) Close() {
	if r.Pending() {
		gl.DeleteSync(r.fence)
	}
	if r.buffer != 0 {
		gl.DeleteBuffers(1, &r.buffer)
		removeGPUBytes(uint64(r.size))
	}
	*r = FramebufferReadback{}
}
