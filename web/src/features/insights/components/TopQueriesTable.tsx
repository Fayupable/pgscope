import { useState } from 'react'
import type { SlowQuery } from '../../../shared/types/insights'
import { formatDuration } from '../utils/formatDuration'

const PAGE_SIZE = 10

export function TopQueriesTable({ queries }: { queries: SlowQuery[] }) {
    const [page, setPage] = useState(0)
    const [expanded, setExpanded] = useState<Set<number>>(new Set())

    if (queries.length === 0) {
        return <p className="insights-empty">No query statistics available yet.</p>
    }

    const pageCount = Math.ceil(queries.length / PAGE_SIZE)
    const start = page * PAGE_SIZE
    const pageQueries = queries.slice(start, start + PAGE_SIZE)

    function toggleExpanded(index: number) {
        setExpanded((prev) => {
            const next = new Set(prev)
            if (next.has(index)) {
                next.delete(index)
            } else {
                next.add(index)
            }
            return next
        })
    }

    return (
        <>
            <table className="top-queries-table">
                <thead>
                    <tr>
                        <th>Query</th>
                        <th>Calls</th>
                        <th title="Sum of every call's duration — how much total server time this query shape has consumed, not how long it ran just now">
                            Total time spent
                        </th>
                        <th title="Average duration per call — the number that best reflects how slow a single execution actually is">
                            Avg per call
                        </th>
                    </tr>
                </thead>
                <tbody>
                    {pageQueries.map((q, i) => {
                        const isExpanded = expanded.has(start + i)
                        return (
                            <tr key={start + i}>
                                <td
                                    className={
                                        'top-queries-table__query' +
                                        (isExpanded ? ' top-queries-table__query--expanded' : '')
                                    }
                                    onClick={() => toggleExpanded(start + i)}
                                    title={isExpanded ? undefined : 'Click to see the full query'}
                                >
                                    {q.query}
                                </td>
                                <td>{q.calls.toLocaleString()}</td>
                                <td>{formatDuration(q.totalExecMs)}</td>
                                <td>{formatDuration(q.meanExecMs)}</td>
                            </tr>
                        )
                    })}
                </tbody>
            </table>

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