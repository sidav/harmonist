package main

import (
	"sort"

	"github.com/anaseto/gruid"
)

type dungeonPath struct {
	dungeon   *dungeon
	neighbors [8]gruid.Point
	wcost     int
}

func (dp *dungeonPath) Neighbors(pos gruid.Point) []gruid.Point {
	nb := dp.neighbors[:0]
	return CardinalNeighbors(pos, nb, func(npos gruid.Point) bool { return valid(npos) })
}

func (dp *dungeonPath) Cost(from, to gruid.Point) int {
	if terrain(dp.dungeon.Cell(to)) == WallCell {
		if dp.wcost > 0 {
			return dp.wcost
		}
		return 4
	}
	return 1
}

func (dp *dungeonPath) Estimation(from, to gruid.Point) int {
	return Distance(from, to)
}

type gridPath struct {
	dungeon   *dungeon
	neighbors [4]gruid.Point
}

func (gp *gridPath) Neighbors(pos gruid.Point) []gruid.Point {
	nb := gp.neighbors[:0]
	return CardinalNeighbors(pos, nb, func(npos gruid.Point) bool { return valid(npos) })
}

func (gp *gridPath) Cost(from, to gruid.Point) int {
	return 1
}

func (gp *gridPath) Estimation(from, to gruid.Point) int {
	return Distance(from, to)
}

type mappingPath struct {
	state     *game
	neighbors [8]gruid.Point
}

func (dp *mappingPath) Neighbors(pos gruid.Point) []gruid.Point {
	d := dp.state.Dungeon
	if terrain(d.Cell(pos)) == WallCell {
		return nil
	}
	nb := dp.neighbors[:0]
	keep := func(npos gruid.Point) bool {
		return valid(npos)
	}
	return CardinalNeighbors(pos, nb, keep)
}

func (dp *mappingPath) Cost(from, to gruid.Point) int {
	return 1
}

func (dp *mappingPath) Estimation(from, to gruid.Point) int {
	return Distance(from, to)
}

type tunnelPath struct {
	dg        *dgen
	neighbors [4]gruid.Point
	area      [9]gruid.Point
}

func (tp *tunnelPath) Neighbors(pos gruid.Point) []gruid.Point {
	nb := tp.neighbors[:0]
	return CardinalNeighbors(pos, nb, func(npos gruid.Point) bool { return valid(npos) })
}

func (tp *tunnelPath) Cost(from, to gruid.Point) int {
	if tp.dg.room[from] && !tp.dg.tunnel[from] {
		return 50
	}
	cost := 1
	c := tp.dg.d.Cell(from)
	if tp.dg.room[from] {
		cost += 7
	} else if !tp.dg.tunnel[from] && terrain(c) != GroundCell {
		cost++
	}
	if c.IsPassable() {
		return cost
	}
	wc := tp.dg.WallAreaCount(tp.area[:0], from, 1)
	return cost + 8 - wc
}

func (tp *tunnelPath) Estimation(from, to gruid.Point) int {
	return Distance(from, to)
}

type playerPath struct {
	state     *game
	neighbors [8]gruid.Point
	goal      gruid.Point
}

func (pp *playerPath) Neighbors(pos gruid.Point) []gruid.Point {
	d := pp.state.Dungeon
	nb := pp.neighbors[:0]
	keep := func(npos gruid.Point) bool {
		t, okT := pp.state.TerrainKnowledge[npos]
		if cld, ok := pp.state.Clouds[npos]; ok && cld == CloudFire && (!okT || t != FoliageCell && t != DoorCell) {
			return false
		}
		return valid(npos) && explored(d.Cell(npos)) && (d.Cell(npos).IsPlayerPassable() && !okT ||
			okT && t.IsPlayerPassable() ||
			pp.state.Player.HasStatus(StatusLevitation) && (t == BarrierCell || t == ChasmCell) ||
			pp.state.Player.HasStatus(StatusDig) && (d.Cell(npos).IsDiggable() && !okT || (okT && t.IsDiggable())))
	}
	nb = CardinalNeighbors(pos, nb, keep)
	sort.Slice(nb, func(i, j int) bool {
		return MaxCardinalDist(nb[i], pp.goal) <= MaxCardinalDist(nb[j], pp.goal)
	})
	return nb
}

func (pp *playerPath) Cost(from, to gruid.Point) int {
	if !pp.state.ExclusionsMap[from] && pp.state.ExclusionsMap[to] {
		return unreachable
	}
	return 1
}

func (pp *playerPath) Estimation(from, to gruid.Point) int {
	return Distance(from, to)
}

type jumpPath struct {
	state     *game
	neighbors [8]gruid.Point
}

func (jp *jumpPath) Neighbors(pos gruid.Point) []gruid.Point {
	nb := jp.neighbors[:0]
	keep := func(npos gruid.Point) bool {
		return jp.state.PlayerCanPass(npos)
	}
	nb = CardinalNeighbors(pos, nb, keep)
	nb = ShufflePos(nb)
	return nb
}

func (jp *jumpPath) Cost(from, to gruid.Point) int {
	return 1
}

func (jp *jumpPath) Estimation(from, to gruid.Point) int {
	return Distance(from, to)
}

type noisePath struct {
	state     *game
	neighbors [8]gruid.Point
}

func (fp *noisePath) Neighbors(pos gruid.Point) []gruid.Point {
	nb := fp.neighbors[:0]
	d := fp.state.Dungeon
	keep := func(npos gruid.Point) bool {
		return valid(npos) && terrain(d.Cell(npos)) != WallCell
	}
	return CardinalNeighbors(pos, nb, keep)
}

func (fp *noisePath) Cost(from, to gruid.Point) int {
	return 1
}

type autoexplorePath struct {
	state     *game
	neighbors [8]gruid.Point
}

func (ap *autoexplorePath) Neighbors(pos gruid.Point) []gruid.Point {
	if ap.state.ExclusionsMap[pos] {
		return nil
	}
	d := ap.state.Dungeon
	nb := ap.neighbors[:0]
	keep := func(npos gruid.Point) bool {
		t, okT := ap.state.TerrainKnowledge[npos]
		if cld, ok := ap.state.Clouds[npos]; ok && cld == CloudFire && (!okT || t != FoliageCell && t != DoorCell) {
			// XXX little info leak
			return false
		}
		if !valid(npos) {
			return false
		}
		c := d.Cell(npos)
		return c.IsPlayerPassable() && (!okT && !c.IsWall() || !t.IsWall()) &&
			!ap.state.ExclusionsMap[npos]
	}
	nb = CardinalNeighbors(pos, nb, keep)
	return nb
}

func (ap *autoexplorePath) Cost(from, to gruid.Point) int {
	return 1
}

type monPath struct {
	state     *game
	monster   *monster
	neighbors [8]gruid.Point
}

func ShufflePos(ps []gruid.Point) []gruid.Point {
	for i := 0; i < len(ps); i++ {
		j := i + RandInt(len(ps)-i)
		ps[i], ps[j] = ps[j], ps[i]
	}
	return ps
}

func (mp *monPath) Neighbors(pos gruid.Point) []gruid.Point {
	nb := mp.neighbors[:0]
	keep := func(npos gruid.Point) bool {
		return mp.monster.CanPassDestruct(mp.state, npos)
	}
	ret := CardinalNeighbors(pos, nb, keep)
	// shuffle so that monster movement is not unnaturally predictable
	ret = ShufflePos(ret)
	return ret
}

func (mp *monPath) Cost(from, to gruid.Point) int {
	g := mp.state
	mons := g.MonsterAt(to)
	if !mons.Exists() {
		c := g.Dungeon.Cell(to)
		if mp.monster.Kind == MonsEarthDragon && c.IsDestructible() && !mp.monster.Status(MonsConfused) {
			return 5
		}
		if to == g.Player.Pos && mp.monster.Kind.Peaceful() {
			switch mp.monster.Kind {
			case MonsEarthDragon:
				return 1
			default:
				return 4
			}
		}
		if mp.monster.Kind.Patrolling() && mp.monster.State != Hunting && !c.IsNormalPatrolWay() {
			return 4
		}
		return 1
	}
	if mons.Status(MonsLignified) {
		return 8
	}
	return 6
}

func (mp *monPath) Estimation(from, to gruid.Point) int {
	return Distance(from, to)
}

func (m *monster) APath(g *game, from, to gruid.Point) []gruid.Point {
	mp := &monPath{state: g, monster: m}
	path := g.PR.AstarPath(mp, from, to)
	if len(path) == 0 {
		return nil
	}
	return path
}

func (g *game) PlayerPath(from, to gruid.Point) []gruid.Point {
	pp := &playerPath{state: g, goal: to}
	path := g.PR.AstarPath(pp, from, to)
	if len(path) == 0 {
		return nil
	}
	return path
}

func (g *game) SortedNearestTo(cells []gruid.Point, to gruid.Point) []gruid.Point {
	ps := posSlice{}
	for _, pos := range cells {
		pp := &dungeonPath{dungeon: g.Dungeon, wcost: unreachable}
		path := g.PR.AstarPath(pp, pos, to)
		if len(path) > 0 {
			ps = append(ps, posCost{pos, len(path)})
		}
	}
	sort.Sort(ps)
	sorted := []gruid.Point{}
	for _, pc := range ps {
		sorted = append(sorted, pc.pos)
	}
	return sorted
}

type posCost struct {
	pos  gruid.Point
	cost int
}

type posSlice []posCost

func (ps posSlice) Len() int           { return len(ps) }
func (ps posSlice) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }
func (ps posSlice) Less(i, j int) bool { return ps[i].cost < ps[j].cost }
