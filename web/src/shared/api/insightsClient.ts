import type { Insights } from '../types/insights'

// The wire shape as the Go backend actually serializes it. Every array
// field the MySQL adapter deliberately leaves unpopulated (see
// mysql.InsightsCollector's doc comment on the backend) is a nil Go slice,
// which encodes as JSON `null`, not `[]`. Postgres always populates every
// field, so this only bites when talking to a MySQL-backed server — but
// every consumer of `Insights` should be able to trust every array is a
// real array, never null, regardless of which engine answered. This type
// captures the untrusted wire shape; normalizeInsights below is the one
// place responsible for closing the gap.
type RawInsights = Omit<
    Insights,
    | 'functionCosts'
    | 'invalidIndexes'
    | 'unvalidatedConstraints'
    | 'vacuumHealthWarnings'
    | 'replicationLagWarnings'
    | 'physicalIOHotspots'
    | 'replicationSlotWarnings'
    | 'lockWaitWarnings'
> & {
    functionCosts: Insights['functionCosts'] | null
    invalidIndexes: Insights['invalidIndexes'] | null
    unvalidatedConstraints: Insights['unvalidatedConstraints'] | null
    vacuumHealthWarnings: Insights['vacuumHealthWarnings'] | null
    replicationLagWarnings: Insights['replicationLagWarnings'] | null
    physicalIOHotspots: Insights['physicalIOHotspots'] | null
    replicationSlotWarnings: Insights['replicationSlotWarnings'] | null
    // Absent entirely (undefined) on a Postgres server, since that engine
    // has never heard of this field — as opposed to the fields above,
    // which Postgres always sends populated but MySQL may null out.
    lockWaitWarnings?: Insights['lockWaitWarnings'] | null
}

function normalizeInsights(raw: RawInsights): Insights {
    return {
        ...raw,
        functionCosts: raw.functionCosts ?? [],
        invalidIndexes: raw.invalidIndexes ?? [],
        unvalidatedConstraints: raw.unvalidatedConstraints ?? [],
        vacuumHealthWarnings: raw.vacuumHealthWarnings ?? [],
        replicationLagWarnings: raw.replicationLagWarnings ?? [],
        physicalIOHotspots: raw.physicalIOHotspots ?? [],
        replicationSlotWarnings: raw.replicationSlotWarnings ?? [],
        lockWaitWarnings: raw.lockWaitWarnings ?? [],
    }
}

export async function getInsights(): Promise<Insights> {
    const response = await fetch('/api/v1/insights')
    if (!response.ok) {
        if (response.status === 429) {
            throw new Error('Rate limited — please wait a few seconds before refreshing again.')
        }
        throw new Error(`Failed to fetch insights: ${response.status}`)
    }
    return normalizeInsights(await response.json())
}
