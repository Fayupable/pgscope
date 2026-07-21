import { useState } from 'react'
import type { UnusedIndex } from '../../../shared/types/insights'
import { formatBytes } from '../utils/formatBytes'

const PAGE_SIZE = 10

export function UnusedIndexesTable({ indexes }: { indexes: UnusedIndex[] }) {
    const [page, setPage] = useState(0)

    if (indexes.length === 0) {
        return <p className="insights-empty">No unused indexes detected.</p>
    }

    const pageCount = Math.ceil(indexes.length / PAGE_SIZE)
    const start = page * PAGE_SIZE
    const pageIndexes = indexes.slice(start, start + PAGE_SIZE)

    return (
        <>
            <table className="index-candidates-table">
                <thead>
                    <tr>
                        <th>Table</th>
                        <th>Index</th>
                        <th>Index scans</th>
                        <th>Size</th>
                    </tr>
                </thead>
                <tbody>
                    {pageIndexes.map((idx, i) => (
                        <tr key={start + i}>
                            <td className="index-candidates-table__table">{idx.table}</td>
                            <td>{idx.index}</td>
                            <td>{idx.indexScans.toLocaleString()}</td>
                            <td>{formatBytes(idx.sizeBytes)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>

            <ul className="index-candidates-table__rationales">
                {pageIndexes.map((idx, i) => (
                    <li key={start + i}>
                        <strong>{idx.index}:</strong> {idx.explanation}
                    </li>
                ))}
            </ul>

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