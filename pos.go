package main

import (
	"fmt"
)

func Distance(from, to gruid.Point) int {
	delta := to.Sub(from)
	return Abs(delta.X) + Abs(delta.Y)
}

func MaxCardinalDist(from, to gruid.Point) int {
	delta := to.Sub(from)
	deltaX := Abs(delta.X)
	deltaY := Abs(delta.Y)
	if deltaX > deltaY {
		return deltaX
	}
	return deltaY
}

func DistanceX(from, to gruid.Point) int {
	deltaX := Abs(to.X - from.X)
	return deltaX
}

func DistanceY(from, to gruid.Point) int {
	deltaY := Abs(to.Y - from.Y)
	return deltaY
}

type direction int

const (
	NoDir direction = iota
	E
	ENE
	NE
	NNE
	N
	NNW
	NW
	WNW
	W
	WSW
	SW
	SSW
	S
	SSE
	SE
	ESE
)

func (dir direction) String() (s string) {
	switch dir {
	case NoDir:
		s = ""
	case E:
		s = "E"
	case ENE:
		s = "ENE"
	case NE:
		s = "NE"
	case NNE:
		s = "NNE"
	case N:
		s = "N"
	case NNW:
		s = "NNW"
	case NW:
		s = "NW"
	case WNW:
		s = "WNW"
	case W:
		s = "W"
	case WSW:
		s = "WSW"
	case SW:
		s = "SW"
	case SSW:
		s = "SSW"
	case S:
		s = "S"
	case SSE:
		s = "SSE"
	case SE:
		s = "SE"
	case ESE:
		s = "ESE"
	}
	return s
}

func KeyToDir(k action) (dir direction) {
	switch k {
	case ActionW, ActionRunW:
		dir = W
	case ActionE, ActionRunE:
		dir = E
	case ActionS, ActionRunS:
		dir = S
	case ActionN, ActionRunN:
		dir = N
	}
	return dir
}

func (pos gruid.Point) To(dir direction) gruid.Point {
	to := pos
	switch dir {
	case E, ENE, ESE:
		to = pos.E()
	case NE:
		to = pos.NE()
	case NNE, N, NNW:
		to = pos.N()
	case NW:
		to = pos.NW()
	case WNW, W, WSW:
		to = pos.W()
	case SW:
		to = pos.SW()
	case SSW, S, SSE:
		to = pos.S()
	case SE:
		to = pos.SE()
	}
	return to
}

func (pos gruid.Point) Dir(from gruid.Point) direction {
	deltaX := Abs(pos.X - from.X)
	deltaY := Abs(pos.Y - from.Y)
	switch {
	case pos.X > from.X && pos.Y == from.Y:
		return E
	case pos.X > from.X && pos.Y < from.Y:
		switch {
		case deltaX > deltaY:
			return ENE
		case deltaX == deltaY:
			return NE
		default:
			return NNE
		}
	case pos.X == from.X && pos.Y < from.Y:
		return N
	case pos.X < from.X && pos.Y < from.Y:
		switch {
		case deltaY > deltaX:
			return NNW
		case deltaX == deltaY:
			return NW
		default:
			return WNW
		}
	case pos.X < from.X && pos.Y == from.Y:
		return W
	case pos.X < from.X && pos.Y > from.Y:
		switch {
		case deltaX > deltaY:
			return WSW
		case deltaX == deltaY:
			return SW
		default:
			return SSW
		}
	case pos.X == from.X && pos.Y > from.Y:
		return S
	case pos.X > from.X && pos.Y > from.Y:
		switch {
		case deltaY > deltaX:
			return SSE
		case deltaX == deltaY:
			return SE
		default:
			return ESE
		}
	default:
		panic(fmt.Sprintf("internal error: invalid gruid.Point:%+v-%+v", pos, from))
	}
}

func (pos gruid.Point) Parents(from gruid.Point, p []gruid.Point) []gruid.Point {
	switch pos.Dir(from) {
	case E:
		p = append(p, pos.W())
	case ENE:
		p = append(p, pos.W(), pos.SW())
	case NE:
		p = append(p, pos.SW())
	case NNE:
		p = append(p, pos.S(), pos.SW())
	case N:
		p = append(p, pos.S())
	case NNW:
		p = append(p, pos.S(), pos.SE())
	case NW:
		p = append(p, pos.SE())
	case WNW:
		p = append(p, pos.E(), pos.SE())
	case W:
		p = append(p, pos.E())
	case WSW:
		p = append(p, pos.E(), pos.NE())
	case SW:
		p = append(p, pos.NE())
	case SSW:
		p = append(p, pos.N(), pos.NE())
	case S:
		p = append(p, pos.N())
	case SSE:
		p = append(p, pos.N(), pos.NW())
	case SE:
		p = append(p, pos.NW())
	case ESE:
		p = append(p, pos.W(), pos.NW())
	}
	return p
}

func (pos gruid.Point) RandomNeighbor(diag bool) gruid.Point {
	if diag {
		return pos.RandomNeighborDiagonals()
	}
	return pos.RandomNeighborCardinal()
}

func (pos gruid.Point) RandomNeighborDiagonals() gruid.Point {
	neighbors := [8]gruid.Point{pos.E(), pos.W(), pos.N(), pos.S(), pos.NE(), pos.NW(), pos.SE(), pos.SW()}
	var r int
	switch RandInt(8) {
	case 0:
		r = RandInt(len(neighbors[0:4]))
	case 1:
		r = RandInt(len(neighbors[0:2]))
	default:
		r = RandInt(len(neighbors[4:]))
	}
	return neighbors[r]
}

func (pos gruid.Point) RandomNeighborCardinal() gruid.Point {
	neighbors := [4]gruid.Point{pos.E(), pos.W(), pos.N(), pos.S()}
	var r int
	switch RandInt(4) {
	case 0, 1:
		r = RandInt(len(neighbors[0:2]))
	default:
		r = RandInt(len(neighbors))
	}
	return neighbors[r]
}

func idxtopos(i int) gruid.Point {
	return gruid.Point{i % DungeonWidth, i / DungeonWidth}
}

func (pos gruid.Point) idx() int {
	return pos.Y*DungeonWidth + pos.X
}

func (pos gruid.Point) valid() bool {
	return pos.Y >= 0 && pos.Y < DungeonHeight && pos.X >= 0 && pos.X < DungeonWidth
}

func (pos gruid.Point) Laterals(dir direction) []gruid.Point {
	switch dir {
	case E, ENE, ESE:
		return []gruid.Point{pos.NE(), pos.SE()}
	case NE:
		return []gruid.Point{pos.E(), pos.N()}
	case N, NNE, NNW:
		return []gruid.Point{pos.NW(), pos.NE()}
	case NW:
		return []gruid.Point{pos.W(), pos.N()}
	case W, WNW, WSW:
		return []gruid.Point{pos.SW(), pos.NW()}
	case SW:
		return []gruid.Point{pos.W(), pos.S()}
	case S, SSW, SSE:
		return []gruid.Point{pos.SW(), pos.SE()}
	case SE:
		return []gruid.Point{pos.S(), pos.E()}
	default:
		// should not happen
		return []gruid.Point{}
	}
}

func (dir direction) InViewCone(from, to gruid.Point) bool {
	if to == from {
		return true
	}
	d := to.Dir(from)
	if d == dir || Distance(from, to) <= 1 {
		return true
	}
	switch dir {
	case E:
		switch d {
		case ESE, ENE, NE, SE:
			return true
		}
	case NE:
		switch d {
		case ENE, NNE, N, E:
			return true
		}
	case N:
		switch d {
		case NNE, NNW, NE, NW:
			return true
		}
	case NW:
		switch d {
		case NNW, WNW, N, W:
			return true
		}
	case W:
		switch d {
		case WNW, WSW, NW, SW:
			return true
		}
	case SW:
		switch d {
		case WSW, SSW, W, S:
			return true
		}
	case S:
		switch d {
		case SSW, SSE, SW, SE:
			return true
		}
	case SE:
		switch d {
		case SSE, ESE, S, E:
			return true
		}
	}
	return false
}

var alternateDirs = []direction{E, NE, N, NW, W, SW, S, SE}

func (dir direction) Left() (d direction) {
	switch dir {
	case E:
		d = NE
	case NE:
		d = N
	case N:
		d = NW
	case NW:
		d = W
	case W:
		d = SW
	case SW:
		d = S
	case S:
		d = SE
	case SE:
		d = E
	default:
		d = alternateDirs[RandInt(len(alternateDirs))]
	}
	return d
}

func (dir direction) Right() (d direction) {
	switch dir {
	case E:
		d = SE
	case NE:
		d = E
	case N:
		d = NE
	case NW:
		d = N
	case W:
		d = NW
	case SW:
		d = W
	case S:
		d = SW
	case SE:
		d = S
	default:
		d = alternateDirs[RandInt(len(alternateDirs))]
	}
	return d
}
