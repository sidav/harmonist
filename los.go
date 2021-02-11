// This file implements a line of sight algorithm.
//
// It works in a way that can remind of the Dijkstra algorithm, but within each
// cone between a diagonal and an orthogonal line, only movements along those
// two directions are allowed. This allows the algorithm to be a simple pass on
// squares around the player, starting from radius 1 until line of sight range.
//
// Going from a gruid.Point from to a gruid.Point pos has a cost, which depends
// essentially on the type of terrain in from. Some circumstances, such as
// being on top of a tree, can influence the cost of terrains.
//
// The obtained light rays are lines formed using at most two adjacent
// directions: a diagonal and an orthogonal one (for example north east and
// east).

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/rl"
)

type raynode struct {
	Cost int
}

type rayMap map[gruid.Point]raynode

type lighter struct {
	rs raystyle
	g  *game
}

func (lt *lighter) Cost(src, from, to gruid.Point) int {
	g := lt.g
	rs := lt.rs
	wallcost := lt.MaxCost(src)
	// diagonal costs
	if g.DiagonalOpaque(from, to, rs) {
		return wallcost
	}
	// no terrain cost on origin
	if src == from {
		if rs != TreePlayerRay && g.DiagonalDifficult(from, to) {
			return wallcost - 1
		}
		return Distance(to, from)
	}
	// from terrain specific costs
	c := g.Dungeon.Cell(from)
	if terrain(c) == WallCell {
		return wallcost
	}
	if _, ok := g.Clouds[from]; ok {
		return wallcost
	}
	if terrain(c) == DoorCell {
		if from != src {
			mons := g.MonsterAt(from)
			if !mons.Exists() && from != g.Player.Pos {
				return wallcost
			}
		}
	}
	if terrain(c) == FoliageCell || terrain(c) == HoledWallCell {
		switch rs {
		case TreePlayerRay:
			if terrain(c) == FoliageCell {
				break
			}
			fallthrough
		default:
			return wallcost + Distance(to, from) - 3
		}
	}
	if rs != TreePlayerRay && g.DiagonalDifficult(from, to) {
		cost := wallcost - Distance(from, src) - 1
		if cost < 1 {
			cost = 1
		}
		return cost
	}
	if rs == TreePlayerRay && terrain(c) == WindowCell && Distance(src, from) >= DefaultLOSRange {
		return wallcost - Distance(src, from) - 1
	}
	return Distance(to, from)
}

func (lt *lighter) MaxCost(src gruid.Point) int {
	switch lt.rs {
	case TreePlayerRay:
		return TreeRange + 1
	case MonsterRay:
		return DefaultMonsterLOSRange + 1
	case LightRay:
		return LightRange
	default:
		return lt.g.LosRange() + 1
	}
}

func (g *game) DiagonalOpaque(from, to gruid.Point, rs raystyle) bool {
	// The state uses cardinal movement only, so two diagonal walls should,
	// for example, block line of sight. This is in contrast with the main
	// mechanics of the line of sight algorithm, which for gameplay reasons
	// allows diagonals for light rays in normal circumstances.
	var cache [2]gruid.Point
	p := cache[:0]
	switch Dir(from, to) {
	case NE:
		p = append(p, to.Add(gruid.Point{0, 1}), to.Add(gruid.Point{-1, 0}))
	case NW:
		p = append(p, to.Add(gruid.Point{0, 1}), to.Add(gruid.Point{1, 0}))
	case SW:
		p = append(p, to.Add(gruid.Point{0, -1}), to.Add(gruid.Point{1, 0}))
	case SE:
		p = append(p, to.Add(gruid.Point{0, -1}), to.Add(gruid.Point{-1, 0}))
	}
	count := 0
	for _, pos := range p {
		_, ok := g.Clouds[pos]
		if ok {
			count++
			continue
		}
		if !valid(pos) {
			continue
		}
		c := g.Dungeon.Cell(pos)
		switch terrain(c) {
		case WallCell, HoledWallCell, WindowCell:
			count++
		}
	}
	return count > 1
}

func (g *game) DiagonalDifficult(from, to gruid.Point) bool {
	// For reasons similar as in DiagonalOpaque, two diagonal foliage cells
	// should reduce range of line of sight in that diagonal direction.
	var cache [2]gruid.Point
	p := cache[:0]
	switch Dir(from, to) {
	case NE:
		p = append(p, to.Add(gruid.Point{0, 1}), to.Add(gruid.Point{-1, 0}))
	case NW:
		p = append(p, to.Add(gruid.Point{0, 1}), to.Add(gruid.Point{1, 0}))
	case SW:
		p = append(p, to.Add(gruid.Point{0, -1}), to.Add(gruid.Point{1, 0}))
	case SE:
		p = append(p, to.Add(gruid.Point{0, -1}), to.Add(gruid.Point{-1, 0}))
	}
	count := 0
	for _, pos := range p {
		if !valid(pos) {
			continue
		}
		_, ok := g.Clouds[pos]
		if ok {
			count++
			continue
		}
		switch terrain(g.Dungeon.Cell(pos)) {
		case WallCell, FoliageCell, HoledWallCell:
			count++
		}
	}
	return count > 1
}

type raystyle int

const (
	NormalPlayerRay raystyle = iota
	MonsterRay
	TreePlayerRay
	LightRay
)

const LightRange = 6

const DefaultLOSRange = 12
const DefaultMonsterLOSRange = 12

func (g *game) LosRange() int {
	return DefaultLOSRange
}

func (g *game) StopAuto() {
	if g.Autoexploring && !g.AutoHalt {
		g.Print("You stop exploring.")
	} else if g.AutoDir != NoDir {
		g.Print("You stop.")
	} else if g.AutoTarget != InvalidPos {
		g.Print("You stop.")
	}
	g.AutoHalt = true
	g.AutoDir = NoDir
	g.AutoTarget = InvalidPos
}

const TreeRange = 50

func (g *game) Illuminated(p gruid.Point) bool {
	c, ok := g.LightFOV.At(p)
	return ok && c <= LightRange
}

func (g *game) ComputeLOS() {
	g.ComputeLights()
	for k := range g.Player.LOS {
		delete(g.Player.LOS, k)
	}
	c := g.Dungeon.Cell(g.Player.Pos)
	rs := NormalPlayerRay
	if terrain(c) == TreeCell {
		rs = TreePlayerRay
	}
	lt := &lighter{rs: rs, g: g}
	lnodes := g.Player.FOV.VisionMap(lt, g.Player.Pos)
	nb := make([]gruid.Point, 8)
	for _, n := range lnodes {
		if n.Cost <= DefaultLOSRange {
			g.Player.LOS[n.P] = true
		} else if terrain(c) == TreeCell && g.Illuminated(n.P) && n.Cost <= TreeRange {
			if terrain(g.Dungeon.Cell(n.P)) == WallCell {
				// this is just an approximation, but ok in practice
				nb = Neighbors(n.P, nb, func(npos gruid.Point) bool {
					if !valid(npos) || !g.Illuminated(npos) || g.Dungeon.Cell(npos).IsWall() {
						return false
					}
					cost, ok := g.Player.FOV.At(npos)
					return ok && cost < TreeRange
				})

				if len(nb) == 0 {
					continue
				}
			}
			g.Player.LOS[n.P] = true
		}
	}
	for pos := range g.Player.LOS {
		if g.Player.Sees(pos) {
			g.SeePosition(pos)
		}
	}
	for _, mons := range g.Monsters {
		if mons.Exists() && g.Player.Sees(mons.Pos) {
			mons.ComputeLOS(g) // approximation of what the monster will see for player info purposes
			mons.UpdateKnowledge(g, mons.Pos)
			if mons.Seen {
				g.StopAuto()
				continue
			}
			mons.Seen = true
			g.Printf("You see %s (%v).", mons.Kind.Indefinite(false), mons.State)
			if mons.Kind.Notable() {
				g.StoryPrintf("Saw %s", mons.Kind)
			}
			g.StopAuto()
		}
	}
}

func (m *monster) ComputeLOS(g *game) {
	if m.Kind.Peaceful() {
		return
	}
	for k := range m.LOS {
		delete(m.LOS, k)
	}
	if g.mfov == nil {
		g.mfov = rl.NewFOV(gruid.NewRange(0, 0, DungeonWidth, DungeonHeight))
	}
	losRange := DefaultMonsterLOSRange
	lt := &lighter{rs: MonsterRay, g: g}
	lnodes := g.mfov.VisionMap(lt, m.Pos)
	for _, n := range lnodes {
		if n.P == m.Pos {
			m.LOS[n.P] = true
			continue
		}
		if n.Cost <= losRange && terrain(g.Dungeon.Cell(n.P)) != BarrelCell {
			pnode, ok := g.mfov.From(lt, n.P)
			if !ok || !g.Dungeon.Cell(pnode.P).Hides() {
				m.LOS[n.P] = true
			}
		}
	}
}

func (g *game) SeeNotable(c cell, pos gruid.Point) {
	switch terrain(c) {
	case MagaraCell:
		mag := g.Objects.Magaras[pos]
		dp := &mappingPath{state: g}
		path := g.PR.AstarPath(dp, g.Player.Pos, pos)
		if len(path) > 0 {
			g.StoryPrintf("Spotted %s (distance: %d)", mag, len(path))
		} else {
			g.StoryPrintf("Spotted %s", mag)
		}
	case ItemCell:
		it := g.Objects.Items[pos]
		dp := &mappingPath{state: g}
		path := g.PR.AstarPath(dp, g.Player.Pos, pos)
		if len(path) > 0 {
			g.StoryPrintf("Spotted %s (distance: %d)", it.ShortDesc(g), len(path))
		} else {
			g.StoryPrintf("Spotted %s", it.ShortDesc(g))
		}
	case StairCell:
		st := g.Objects.Stairs[pos]
		dp := &mappingPath{state: g}
		path := g.PR.AstarPath(dp, g.Player.Pos, pos)
		if len(path) > 0 {
			g.StoryPrintf("Discovered %s (distance: %d)", st, len(path))
		} else {
			g.StoryPrintf("Discovered %s", st)
		}
	case FakeStairCell:
		dp := &mappingPath{state: g}
		path := g.PR.AstarPath(dp, g.Player.Pos, pos)
		if len(path) > 0 {
			g.StoryPrintf("Discovered %s (distance: %d)", NormalStairShortDesc, len(path))
		} else {
			g.StoryPrintf("Discovered %s", NormalStairShortDesc)
		}
	case StoryCell:
		st := g.Objects.Story[pos]
		if st == StoryArtifactSealed {
			dp := &mappingPath{state: g}
			path := g.PR.AstarPath(dp, g.Player.Pos, pos)
			if len(path) > 0 {
				g.StoryPrintf("Discovered Portal Moon Gem Artifact (distance: %d)", len(path))
			} else {
				g.StoryPrint("Discovered Portal Moon Gem Artifact")
			}
		}
	}
}

func (g *game) SeePosition(pos gruid.Point) {
	c := g.Dungeon.Cell(pos)
	t, okT := g.TerrainKnowledge[pos]
	if !explored(c) {
		see := "see"
		if c.IsNotable() {
			g.Printf("You %s %s.", see, c.ShortDesc(g, pos))
			g.StopAuto()
		}
		g.Dungeon.SetExplored(pos)
		g.SeeNotable(c, pos)
		g.AutoexploreMapRebuild = true
	} else {
		// XXX this can be improved to handle more terrain types changes
		if okT && t == WallCell && terrain(c) != WallCell {
			g.Printf("There is no longer a wall there.")
			g.StopAuto()
			g.AutoexploreMapRebuild = true
		}
		if cld, ok := g.Clouds[pos]; ok && cld == CloudFire && okT && (t == FoliageCell || t == DoorCell) {
			g.Printf("There are flames there.")
			g.StopAuto()
			g.AutoexploreMapRebuild = true
		}
	}
	if okT {
		delete(g.TerrainKnowledge, pos)
		if c.IsPlayerPassable() {
			delete(g.MagicalBarriers, pos)
		}
	}
	if mons, ok := g.LastMonsterKnownAt[pos]; ok && (mons.Pos != pos || !mons.Exists()) {
		delete(g.LastMonsterKnownAt, pos)
		mons.LastKnownPos = InvalidPos
	}
	delete(g.NoiseIllusion, pos)
	if g.Objects.Story[pos] == StoryShaedra && !g.LiberatedShaedra &&
		(Distance(g.Player.Pos, pos) <= 1 ||
			Distance(g.Player.Pos, g.Places.Marevor) <= 1 ||
			Distance(g.Player.Pos, g.Places.Monolith) <= 1) &&
		g.Player.Pos != g.Places.Marevor &&
		g.Player.Pos != g.Places.Monolith {
		g.PushEventFirst(&playerEvent{EAction: StorySequence}, g.Turn)
		g.LiberatedShaedra = true
	}
}

func (g *game) ComputeExclusion(pos gruid.Point, toggle bool) {
	exclusionRange := g.LosRange()
	g.ExclusionsMap[pos] = toggle
	for d := 1; d <= exclusionRange; d++ {
		for x := -d + pos.X; x <= d+pos.X; x++ {
			for _, pos := range []gruid.Point{{x, pos.Y + d}, {x, pos.Y - d}} {
				if !valid(pos) {
					continue
				}
				g.ExclusionsMap[pos] = toggle
			}
		}
		for y := -d + 1 + pos.Y; y <= d-1+pos.Y; y++ {
			for _, pos := range []gruid.Point{{pos.X + d, y}, {pos.X - d, y}} {
				if !valid(pos) {
					continue
				}
				g.ExclusionsMap[pos] = toggle
			}
		}
	}
}

func (g *game) Ray(p gruid.Point) []gruid.Point {
	c := g.Dungeon.Cell(g.Player.Pos)
	rs := NormalPlayerRay
	if terrain(c) == TreeCell {
		rs = TreePlayerRay
	}
	lt := &lighter{rs: rs, g: g}
	lnodes := g.Player.FOV.Ray(lt, p)
	ps := []gruid.Point{}
	for i := len(lnodes) - 1; i > 0; i-- {
		ps = append(ps, lnodes[i].P)
	}
	return ps
}

//func (g *game) ComputeRayHighlight(pos gruid.Point) {
//g.Highlight = map[gruid.Point]bool{}
//ray := g.Ray(pos)
//for _, p := range ray {
//g.Highlight[p] = true
//}
//}

func (g *game) ComputeNoise() {
	dij := &noisePath{state: g}
	rg := DefaultLOSRange
	nodes := g.PR.BreadthFirstMap(dij, []gruid.Point{g.Player.Pos}, rg)
	count := 0
	for k := range g.Noise {
		delete(g.Noise, k)
	}
	rmax := 2
	if g.Player.Inventory.Body == CloakHear {
		rmax += 2
	}
	for _, n := range nodes {
		if g.Player.Sees(n.P) {
			continue
		}
		mons := g.MonsterAt(n.P)
		if mons.Exists() && mons.State != Resting && mons.State != Watching &&
			(RandInt(rmax) > 0 || terrain(g.Dungeon.Cell(mons.Pos)) == QueenRockCell) {
			switch mons.Kind {
			case MonsMirrorSpecter, MonsSatowalgaPlant, MonsButterfly:
				if mons.Kind == MonsMirrorSpecter && g.Player.Inventory.Body == CloakHear {
					g.Noise[n.P] = true
					g.Print("You hear an imperceptible air movement.")
					count++
				}
			case MonsWingedMilfid, MonsTinyHarpy:
				g.Noise[n.P] = true
				g.Print("You hear the flapping of wings.")
				count++
			case MonsEarthDragon, MonsTreeMushroom, MonsYack:
				g.Noise[n.P] = true
				g.Print("You hear heavy footsteps.")
				count++
			case MonsWorm, MonsAcidMound:
				g.Noise[n.P] = true
				g.Print("You hear a creep noise.")
				count++
			case MonsDog, MonsBlinkingFrog, MonsHazeCat, MonsCrazyImp, MonsSpider:
				g.Noise[n.P] = true
				g.Print("You hear light footsteps.")
				count++
			default:
				g.Noise[n.P] = true
				g.Print("You hear footsteps.")
				count++
			}
		}
	}
	if count > 0 {
		g.StopAuto()
	}
}

func (p *player) Sees(pos gruid.Point) bool {
	//return pos == p.Pos || p.LOS[pos] && p.Dir.InViewCone(p.Pos, pos)
	return p.LOS[pos]
}

func (m *monster) SeesPlayer(g *game) bool {
	return m.Sees(g, g.Player.Pos) && g.Player.Sees(m.Pos)
}

func (m *monster) SeesLight(g *game, pos gruid.Point) bool {
	if !(m.LOS[pos] && m.Dir.InViewCone(m.Pos, pos)) {
		return false
	}
	if m.State == Resting && Distance(m.Pos, pos) > 1 {
		return false
	}
	return true
}

func (m *monster) Sees(g *game, pos gruid.Point) bool {
	var darkRange = 4
	if m.Kind == MonsHazeCat {
		darkRange = DefaultMonsterLOSRange
	}
	if g.Player.Inventory.Body == CloakShadows {
		darkRange--
	}
	if g.Player.HasStatus(StatusShadows) {
		darkRange = 1
	}
	const tableRange = 1
	if !(m.LOS[pos] && (m.Dir.InViewCone(m.Pos, pos) || m.Kind == MonsSpider)) {
		return false
	}
	if m.State == Resting && Distance(m.Pos, pos) > 1 {
		return false
	}
	c := g.Dungeon.Cell(pos)
	if (!g.Illuminated(pos) && !g.Player.HasStatus(StatusIlluminated) || !c.IsIlluminable()) && Distance(m.Pos, pos) > darkRange {
		return false
	}
	if terrain(c) == TableCell && Distance(m.Pos, pos) > tableRange {
		return false
	}
	if g.Player.HasStatus(StatusTransparent) && g.Illuminated(pos) && Distance(m.Pos, pos) > 1 {
		return false
	}
	return true
}

func (g *game) ComputeMonsterLOS() {
	for k := range g.MonsterLOS {
		delete(g.MonsterLOS, k)
	}
	for _, mons := range g.Monsters {
		if !mons.Exists() || !g.Player.Sees(mons.Pos) {
			continue
		}
		for pos := range g.Player.LOS {
			if !g.Player.Sees(pos) {
				continue
			}
			if mons.Sees(g, pos) {
				g.MonsterLOS[pos] = true
			}
		}
	}
	if g.MonsterLOS[g.Player.Pos] {
		g.Player.Statuses[StatusUnhidden] = 1
		g.Player.Statuses[StatusHidden] = 0
	} else {
		g.Player.Statuses[StatusUnhidden] = 0
		g.Player.Statuses[StatusHidden] = 1
	}
	if g.Illuminated(g.Player.Pos) && g.Dungeon.Cell(g.Player.Pos).IsIlluminable() {
		g.Player.Statuses[StatusLight] = 1
	} else {
		g.Player.Statuses[StatusLight] = 0
	}
}

func (g *game) ComputeLights() {
	if g.LightFOV == nil {
		g.LightFOV = rl.NewFOV(gruid.NewRange(0, 0, DungeonWidth, DungeonHeight))
	}
	sources := []gruid.Point{}
	for lpos, on := range g.Objects.Lights {
		if !on {
			continue
		}
		if Distance(lpos, g.Player.Pos) > DefaultLOSRange+LightRange && terrain(g.Dungeon.Cell(g.Player.Pos)) != TreeCell {
			continue
		}
		sources = append(sources, lpos)
	}
	for _, mons := range g.Monsters {
		if !mons.Exists() || mons.Kind != MonsButterfly || mons.Status(MonsConfused) || mons.Status(MonsParalysed) {
			continue
		}
		if Distance(mons.Pos, g.Player.Pos) > DefaultLOSRange+LightRange && terrain(g.Dungeon.Cell(g.Player.Pos)) != TreeCell {
			continue
		}
		sources = append(sources, mons.Pos)
	}
	lt := &lighter{rs: LightRay, g: g}
	g.LightFOV.LightMap(lt, sources)
}

func (g *game) ComputeMonsterCone(m *monster) {
	g.MonsterTargLOS = make(map[gruid.Point]bool)
	for pos := range g.Player.LOS {
		if !g.Player.Sees(pos) {
			continue
		}
		if m.Sees(g, pos) {
			g.MonsterTargLOS[pos] = true
		}
	}
}

func (m *monster) UpdateKnowledge(g *game, pos gruid.Point) {
	if mons, ok := g.LastMonsterKnownAt[pos]; ok {
		mons.LastKnownPos = InvalidPos
	}
	if m.LastKnownPos != InvalidPos {
		delete(g.LastMonsterKnownAt, m.LastKnownPos)
	}
	g.LastMonsterKnownAt[pos] = m
	m.LastSeenState = m.State
	m.LastKnownPos = pos
}
