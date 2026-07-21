import { boundingBoxOf } from './geometry'
import type { Position } from './types'

export const CANVAS_WIDTH = 1400
const COMPONENT_PADDING = 140

export function packComponents(
    componentLayouts: { ids: string[]; positions: Map<string, Position> }[],
): Map<string, Position> {
    const result = new Map<string, Position>()

    let cursorX = 0
    let cursorY = 0
    let rowHeight = 0

    for (const component of componentLayouts) {
        const box = boundingBoxOf(component.positions)
        const width = box.maxX - box.minX
        const height = box.maxY - box.minY

        if (cursorX + width > CANVAS_WIDTH && cursorX > 0) {
            cursorX = 0
            cursorY += rowHeight + COMPONENT_PADDING
            rowHeight = 0
        }

        const offsetX = cursorX - box.minX
        const offsetY = cursorY - box.minY

        for (const [id, pos] of component.positions) {
            result.set(id, { x: pos.x + offsetX, y: pos.y + offsetY })
        }

        cursorX += width + COMPONENT_PADDING
        rowHeight = Math.max(rowHeight, height)
    }

    return result
}

export { COMPONENT_PADDING }