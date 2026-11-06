package generator

import "math"

// pnxNoise is a deterministic gradient-noise implementation used by the PNX-style
// density sampler. It deliberately has no mutable state, so chunk generation is
// safe when Dragonfly runs multiple chunk workers concurrently.

func mix64(v uint64) uint64 {
	v ^= v >> 30
	v *= 0xbf58476d1ce4e5b9
	v ^= v >> 27
	v *= 0x94d049bb133111eb
	v ^= v >> 31
	return v
}

func latticeHash(seed int64, x, y, z int64) uint64 {
	h := uint64(seed) ^ 0x9e3779b97f4a7c15
	h = mix64(h ^ uint64(x)*0x632be59bd9b4e019)
	h = mix64(h ^ uint64(y)*0x8cb92baa3f3d8dd7)
	h = mix64(h ^ uint64(z)*0x9e3779b185ebca87)
	return h
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func grad2(hash uint64, x, z float64) float64 {
	switch hash & 7 {
	case 0:
		return x + z
	case 1:
		return -x + z
	case 2:
		return x - z
	case 3:
		return -x - z
	case 4:
		return x
	case 5:
		return -x
	case 6:
		return z
	default:
		return -z
	}
}

func noise2(seed int64, x, z float64) float64 {
	x0, z0 := int64(math.Floor(x)), int64(math.Floor(z))
	xf, zf := x-math.Floor(x), z-math.Floor(z)
	u, v := fade(xf), fade(zf)

	n00 := grad2(latticeHash(seed, x0, 0, z0), xf, zf)
	n10 := grad2(latticeHash(seed, x0+1, 0, z0), xf-1, zf)
	n01 := grad2(latticeHash(seed, x0, 0, z0+1), xf, zf-1)
	n11 := grad2(latticeHash(seed, x0+1, 0, z0+1), xf-1, zf-1)
	return clamp(lerp(lerp(n00, n10, u), lerp(n01, n11, u), v)*0.7071067811865476, -1, 1)
}

func grad3(hash uint64, x, y, z float64) float64 {
	switch hash % 12 {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	case 3:
		return -x - y
	case 4:
		return x + z
	case 5:
		return -x + z
	case 6:
		return x - z
	case 7:
		return -x - z
	case 8:
		return y + z
	case 9:
		return -y + z
	case 10:
		return y - z
	default:
		return -y - z
	}
}

func noise3(seed int64, x, y, z float64) float64 {
	xFloor, yFloor, zFloor := math.Floor(x), math.Floor(y), math.Floor(z)
	x0, y0, z0 := int64(xFloor), int64(yFloor), int64(zFloor)
	xf, yf, zf := x-xFloor, y-yFloor, z-zFloor
	u, v, w := fade(xf), fade(yf), fade(zf)

	c000 := grad3(latticeHash(seed, x0, y0, z0), xf, yf, zf)
	c100 := grad3(latticeHash(seed, x0+1, y0, z0), xf-1, yf, zf)
	c010 := grad3(latticeHash(seed, x0, y0+1, z0), xf, yf-1, zf)
	c110 := grad3(latticeHash(seed, x0+1, y0+1, z0), xf-1, yf-1, zf)
	c001 := grad3(latticeHash(seed, x0, y0, z0+1), xf, yf, zf-1)
	c101 := grad3(latticeHash(seed, x0+1, y0, z0+1), xf-1, yf, zf-1)
	c011 := grad3(latticeHash(seed, x0, y0+1, z0+1), xf, yf-1, zf-1)
	c111 := grad3(latticeHash(seed, x0+1, y0+1, z0+1), xf-1, yf-1, zf-1)

	lower := lerp(lerp(c000, c100, u), lerp(c010, c110, u), v)
	upper := lerp(lerp(c001, c101, u), lerp(c011, c111, u), v)
	return clamp(lerp(lower, upper, w)*0.5773502691896258, -1, 1)
}

func fbm2(seed int64, x, z, scale float64, octaves int) float64 {
	frequency, amplitude, total, weight := scale, 1.0, 0.0, 0.0
	for octave := 0; octave < octaves; octave++ {
		total += noise2(seed+int64(octave)*0x1f123bb5, x*frequency, z*frequency) * amplitude
		weight += amplitude
		frequency *= 2.0
		amplitude *= 0.5
	}
	if weight == 0 {
		return 0
	}
	return total / weight
}

// fbm2Normalized expands the compact output range of this gradient-noise
// implementation into the approximately [-1, 1] climate/density range used by
// the overworld's continentalness and climate classifiers.
func fbm2Normalized(seed int64, x, z, scale float64, octaves int) float64 {
	return clamp(fbm2(seed, x, z, scale, octaves)*3.5, -1, 1)
}

func fbm3(seed int64, x, y, z, scale float64, octaves int) float64 {
	frequency, amplitude, total, weight := scale, 1.0, 0.0, 0.0
	for octave := 0; octave < octaves; octave++ {
		total += noise3(seed+int64(octave)*0x1f123bb5, x*frequency, y*frequency, z*frequency) * amplitude
		weight += amplitude
		frequency *= 2.0
		amplitude *= 0.5
	}
	if weight == 0 {
		return 0
	}
	return total / weight
}

func clamp(value, minimum, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func smoothstep(edge0, edge1, value float64) float64 {
	if edge0 == edge1 {
		if value < edge0 {
			return 0
		}
		return 1
	}
	t := clamp((value-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}
