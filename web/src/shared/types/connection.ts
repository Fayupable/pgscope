export type Engine = 'postgres' | 'mysql'

export interface Connection {
    id: string
    engine: Engine
}
