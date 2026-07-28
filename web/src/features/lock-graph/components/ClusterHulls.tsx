import { useViewport } from '@xyflow/react'
import type { ClusterHull } from '../../../shared/utils/graph'

const HULL_COLORS = [
  { fill: 'var(--cluster-hull-1)', stroke: 'var(--cluster-hull-stroke-1)' },
  { fill: 'var(--cluster-hull-2)', stroke: 'var(--cluster-hull-stroke-2)' },
  { fill: 'var(--cluster-hull-3)', stroke: 'var(--cluster-hull-stroke-3)' },
  { fill: 'var(--cluster-hull-4)', stroke: 'var(--cluster-hull-stroke-4)' },
  { fill: 'var(--cluster-hull-5)', stroke: 'var(--cluster-hull-stroke-5)' },
]

// Node positions from graphLayout represent each node's top-left corner (to
// match React Flow's own convention), but the hull was computed from those
// same raw coordinates. Offsetting by half the node's rendered size (44px)
// aligns the hull shape with where nodes visually appear (their centers).
const NODE_CENTER_OFFSET = 22

// How far (in px) the rounding cuts into each corner, capped at half the
// adjacent edge's length so it never overreaches on a small polygon. Sized
// generously (much larger than the first attempt at this) so a large,
// many-cornered cluster still reads as one smooth, rounded blob rather
// than a faceted polygon with only its very tips rounded off.
const CORNER_ROUNDING_RADIUS = 50

/**
 * Builds a closed SVG path that rounds each corner of the polygon
 * independently — a quadratic Bezier through the corner itself, anchored
 * at points a fixed radius along each adjacent edge — rather than fitting
 * one smooth curve through every point at once (what a d3 curve like
 * curveCatmullRomClosed or curveCardinalClosed does).
 *
 * That global-fit approach is what previously caused unpredictable
 * failures depending on point count and arrangement: with few, unevenly
 * spaced, or near-collinear points (e.g. a 3-node cluster roughly in a
 * line), a Catmull-Rom/Cardinal spline's tangent estimation can overshoot
 * far beyond the polygon, or (at high tension) pull in so far it no
 * longer reaches some of the original points at all. Rounding each corner
 * locally sidesteps both failure modes by construction: a quadratic
 * Bezier with the corner itself as control point is mathematically
 * guaranteed to stay within the triangle formed by the corner and its two
 * rounding-radius anchor points, so the rendered shape can never extend
 * past the original polygon and can never fail to reach within
 * CORNER_ROUNDING_RADIUS of every original point — regardless of how many
 * points there are or how they're arranged.
 */
function roundedPolygonPath(points: { x: number; y: number }[], radius: number): string {
  const n = points.length
  if (n < 3) return ''

  const segments: string[] = []

  for (let i = 0; i < n; i++) {
    const prev = points[(i - 1 + n) % n]
    const cur = points[i]
    const next = points[(i + 1) % n]

    const toPrev = { x: prev.x - cur.x, y: prev.y - cur.y }
    const toNext = { x: next.x - cur.x, y: next.y - cur.y }
    const distPrev = Math.hypot(toPrev.x, toPrev.y) || 1
    const distNext = Math.hypot(toNext.x, toNext.y) || 1
    const rPrev = Math.min(radius, distPrev / 2)
    const rNext = Math.min(radius, distNext / 2)

    const pre = { x: cur.x + (toPrev.x / distPrev) * rPrev, y: cur.y + (toPrev.y / distPrev) * rPrev }
    const post = { x: cur.x + (toNext.x / distNext) * rNext, y: cur.y + (toNext.y / distNext) * rNext }

    if (i === 0) {
      segments.push(`M ${pre.x} ${pre.y}`)
    } else {
      segments.push(`L ${pre.x} ${pre.y}`)
    }
    segments.push(`Q ${cur.x} ${cur.y} ${post.x} ${post.y}`)
  }

  segments.push('Z')
  return segments.join(' ')
}

interface ClusterHullsProps {
  hulls: ClusterHull[]
}

/**
 * Renders each cluster's convex hull as a shaded region with rounded
 * corners behind the graph's nodes/edges, manually kept in sync with
 * React Flow's pan/zoom via useViewport() — this component must be
 * rendered as a child of <ReactFlow> to access that context.
 */
export function ClusterHulls({ hulls }: ClusterHullsProps) {
  const { x, y, zoom } = useViewport()

  return (
    <svg className="cluster-hulls">
      <g transform={`translate(${x}, ${y}) scale(${zoom})`}>
        {hulls.map((hull, index) => {
          const colors = HULL_COLORS[hull.colorIndex % HULL_COLORS.length]
          const centeredPoints = hull.points.map((p) => ({
            x: p.x + NODE_CENTER_OFFSET,
            y: p.y + NODE_CENTER_OFFSET,
          }))
          const path = roundedPolygonPath(centeredPoints, CORNER_ROUNDING_RADIUS)

          if (!path) return null

          return <path key={index} d={path} fill={colors.fill} stroke={colors.stroke} strokeWidth={1.5} />
        })}
      </g>
    </svg>
  )
}
