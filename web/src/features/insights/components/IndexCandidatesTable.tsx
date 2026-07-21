import { useState } from 'react'
import type { Confidence, IndexCandidate } from '../../../shared/types/insights'

const CONFIDENCE_LABEL: Record<Confidence, string> = {
    strong: 'Strong',
    weak: 'Weak',
    insufficient_data: 'Insufficient data',
}

const PAGE_SIZE = 10

function ConfidenceBadge({ confidence }: { confidence: Confidence }) {
    return <span className={`confidence-badge confidence-badge--${confidence}`}>{CONFIDENCE_LABEL[confidence]}</span>
}

export function IndexCandidatesTable({ candidates }: { candidates: IndexCandidate[] }) {
    const [page, setPage] = useState(0)

    if (candidates.length === 0) {
        return <p className="insights-empty">No index candidates detected.</p>
    }

    const pageCount = Math.ceil(candidates.length / PAGE_SIZE)
    const start = page * PAGE_SIZE
    const pageCandidates = candidates.slice(start, start + PAGE_SIZE)

    return (
        <>
            <table className="index-candidates-table">
                <thead>
                    <tr>
                        <th>Table</th>
                        <th>Confidence</th>
                        <th>Suspected columns</th>
                        <th>Seq scans</th>
                        <th>Index scans</th>
                        <th>Existing indexes</th>
                        <th>Writes/s</th>
                    </tr>
                </thead>
                <tbody>
                    {pageCandidates.map((c) => (
                        <tr key={c.table}>
                            <td className="index-candidates-table__table">{c.table}</td>
                            <td>
                                <ConfidenceBadge confidence={c.confidence} />
                            </td>
                            <td>
                                {c.suspectedColumns.length === 0 ? (
                                    <span className="insights-empty">—</span>
                                ) : (
                                    <div className="index-candidates-table__columns">
                                        {c.suspectedColumns.map((col) => (
                                            <span key={col} className="index-candidate-card__column-chip">
                                                {col}
                                            </span>
                                        ))}
                                    </div>
                                )}
                            </td>
                            <td>{c.seqScanCount.toLocaleString()}</td>
                            <td>{c.idxScanCount.toLocaleString()}</td>
                            <td>{c.existingIndexCount}</td>
                            <td>{c.writesPerSecond.toFixed(2)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>

            <ul className="index-candidates-table__rationales">
                {pageCandidates.map((c) => (
                    <li key={c.table}>
                        <strong>{c.table}:</strong> {c.rationale}
                    </li>
                ))}
            </ul>

            <p className="index-candidates-table__note">
                Suggestion only — a new index speeds up these reads but adds overhead to every insert/update on the
                affected table. Test under your own write load before applying.
            </p>

            {pageCount > 1 && (
                <div className="insights-pagination">
                    <button disabled={page === 0} onClick={() => setPage((p) => p - 1)}>
                        Prev
                    </button>
                    <span>
                        Page {page + 1} of {pageCount}
                    </span>
                    <button disabled={page === pageCount - 1} onClick={() => setPage((p) => p + 1)}>
                        Next
                    </button>
                </div>
            )}
        </>
    )
}