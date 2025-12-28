// Package render3d provides simple 3D rendering for the EZIO-G500 display.
package render3d

import (
	"math"

	"github.com/sagostin/ezio-g500/pkg/eziog500"
)

// Point3D represents a point in 3D space.
type Point3D struct {
	X, Y, Z float64
}

// Point2D represents a point on the 2D screen.
type Point2D struct {
	X, Y int
}

// Camera defines the viewing parameters.
type Camera struct {
	Distance float64 // Distance from origin
	FOV      float64 // Field of view multiplier
	CenterX  int     // Screen center X
	CenterY  int     // Screen center Y
}

// DefaultCamera returns a camera centered on the 128x64 display.
func DefaultCamera() *Camera {
	return &Camera{
		Distance: 4.0,
		FOV:      30.0,
		CenterX:  64,
		CenterY:  32,
	}
}

// Project projects a 3D point onto the 2D screen.
func (c *Camera) Project(p Point3D) Point2D {
	// Simple perspective projection
	z := p.Z + c.Distance
	if z < 0.1 {
		z = 0.1
	}
	x := int(c.FOV*p.X/z) + c.CenterX
	y := int(c.FOV*p.Y/z) + c.CenterY
	return Point2D{X: x, Y: y}
}

// RotateX rotates a point around the X axis.
func RotateX(p Point3D, angle float64) Point3D {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Point3D{
		X: p.X,
		Y: p.Y*cos - p.Z*sin,
		Z: p.Y*sin + p.Z*cos,
	}
}

// RotateY rotates a point around the Y axis.
func RotateY(p Point3D, angle float64) Point3D {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Point3D{
		X: p.X*cos + p.Z*sin,
		Y: p.Y,
		Z: -p.X*sin + p.Z*cos,
	}
}

// RotateZ rotates a point around the Z axis.
func RotateZ(p Point3D, angle float64) Point3D {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Point3D{
		X: p.X*cos - p.Y*sin,
		Y: p.X*sin + p.Y*cos,
		Z: p.Z,
	}
}

// Cube represents a 3D cube with vertices and edges.
type Cube struct {
	Vertices []Point3D
	Edges    [][2]int // Pairs of vertex indices
	Size     float64
}

// NewCube creates a unit cube centered at origin.
func NewCube(size float64) *Cube {
	s := size / 2
	return &Cube{
		Size: size,
		Vertices: []Point3D{
			{-s, -s, -s}, // 0
			{s, -s, -s},  // 1
			{s, s, -s},   // 2
			{-s, s, -s},  // 3
			{-s, -s, s},  // 4
			{s, -s, s},   // 5
			{s, s, s},    // 6
			{-s, s, s},   // 7
		},
		Edges: [][2]int{
			// Front face
			{0, 1}, {1, 2}, {2, 3}, {3, 0},
			// Back face
			{4, 5}, {5, 6}, {6, 7}, {7, 4},
			// Connecting edges
			{0, 4}, {1, 5}, {2, 6}, {3, 7},
		},
	}
}

// Rotate rotates all vertices of the cube.
func (c *Cube) Rotate(angleX, angleY, angleZ float64) {
	for i := range c.Vertices {
		c.Vertices[i] = RotateX(c.Vertices[i], angleX)
		c.Vertices[i] = RotateY(c.Vertices[i], angleY)
		c.Vertices[i] = RotateZ(c.Vertices[i], angleZ)
	}
}

// Draw renders the cube wireframe onto the framebuffer.
func (c *Cube) Draw(fb *eziog500.FrameBuffer, cam *Camera, on bool) {
	// Project all vertices
	projected := make([]Point2D, len(c.Vertices))
	for i, v := range c.Vertices {
		projected[i] = cam.Project(v)
	}

	// Draw all edges
	for _, edge := range c.Edges {
		p1 := projected[edge[0]]
		p2 := projected[edge[1]]
		fb.DrawLine(p1.X, p1.Y, p2.X, p2.Y, on)
	}
}

// CubeCopy returns a copy of the cube for animation.
func (c *Cube) Copy() *Cube {
	newCube := &Cube{
		Size:     c.Size,
		Vertices: make([]Point3D, len(c.Vertices)),
		Edges:    c.Edges, // Edges are shared (immutable indices)
	}
	copy(newCube.Vertices, c.Vertices)
	return newCube
}
