import { createContext, useContext } from 'react'
import type { Engine } from '../types/connection'

// null means "not known yet" — the initial fetch hasn't resolved. Consumers
// that need to hide/show engine-specific UI should treat null the same as
// "don't know, render nothing engine-specific yet" rather than guessing.
export const EngineContext = createContext<Engine | null>(null)

export function useEngine(): Engine | null {
    return useContext(EngineContext)
}
