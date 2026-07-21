import type { Session } from '../../types/session'
import { buildEdges, findBridges, findConnectedComponents, edgesWithin, edgeKey, type GraphEdge } from './edges'
import { layoutComponent } from './clusterLayout'
import { layoutSingletons } from './singletonLayout'
import { packComponents, COMPONENT_PADDING } from './packing'
import { computeClusterHulls, type ClusterHull } from './hulls'
import { boundingBoxOf } from './geometry'
import type { Position } from './types'

export type { Position } from './types'

export interface ClusteredLayout {
    positions: Map<string, Position>
    clusterEdges: GraphEdge[]
    bridgeEdges: GraphEdge[]
    clusterHulls: ClusterHull[]
}

export type { GraphEdge, ClusterHull }

/**
 * Top-level entry point: turns a raw session list into a full layout.
 * Bridge edges are detected and excluded from clustering. Multi-node
 * clusters are laid out independently (hub-centered) and tiled without
 * overlapping. Fully isolated sessions are scattered organically in their
 * own area below the clusters.
 */
export function computeClusteredLayout(sessions: Session[]): ClusteredLayout {
    const nodeIds = sessions.map((s) => s.id)
    const allEdges = buildEdges(sessions)

    if (nodeIds.length === 0) {
        return { positions: new Map(), clusterEdges: [], bridgeEdges: [], clusterHulls: [] }
    }

    const bridges = findBridges(nodeIds, allEdges)
    const bridgeKeys = new Set(bridges.map(edgeKey))
    const clusterEdges = allEdges.filter((e) => !bridgeKeys.has(edgeKey(e)))

    const components = findConnectedComponents(nodeIds, clusterEdges)
    const multiNodeComponents = components.filter((c) => c.length > 1)
    const singletonIds = components.filter((c) => c.length === 1).map((c) => c[0])

    const componentLayouts = multiNodeComponents.map((component) => ({
        ids: component,
        positions: layoutComponent(component, edgesWithin(component, clusterEdges), 0, 0),
    }))

    const packedPositions = packComponents(componentLayouts)
    const singletonStartY = packedPositions.size > 0 ? boundingBoxOf(packedPositions).maxY + COMPONENT_PADDING : 0
    const singletonPositions = layoutSingletons(singletonIds, singletonStartY)

    const positions = new Map([...packedPositions, ...singletonPositions])
    const clusterHulls = computeClusterHulls(componentLayouts, packedPositions)

    return { positions, clusterEdges, bridgeEdges: bridges, clusterHulls }
}