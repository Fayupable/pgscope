import type { Position } from './types'

export function boundingBoxOf(positions: Map<string, Position>): { minX: number; minY: number; maxX: number; maxY: number } {
    let minX = Infinity
    let minY = Infinity
    let maxX = -Infinity
    let maxY = -Infinity

    for (const pos of positions.values()) {
        minX = Math.min(minX, pos.x)
        minY = Math.min(minY, pos.y)
        maxX = Math.max(maxX, pos.x)
        maxY = Math.max(maxY, pos.y)
    }

    if (!Number.isFinite(minX)) return { minX: 0, minY: 0, maxX: 0, maxY: 0 }
    return { minX, minY, maxX, maxY }
}