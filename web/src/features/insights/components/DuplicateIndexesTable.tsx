import { useState } from 'react'
import type { DuplicateIndex } from '../../../shared/types/insights'

const PAGE_SIZE = 10

export function DuplicateIndexesTable({ duplicates }: { duplicates: DuplicateIndex[] }) {
    const [page, setPage] = useState(0)

    if (duplicates.length === 0) {
        return <p className="insights-empty">No redundant indexes detected.</p>
    }

    const pageCount = Math.ceil(duplicates.length / PAGE_SIZE)
    const start = page * PAGE_SIZE
    const pageDuplicates = duplicates.slice(start, start + PAGE_SIZE)

    return (
        <>
            <table className="index-candidates-table">
                <thead>
                    <tr>
                        <th>Table</th>
                        <th>Redundant index</th>
                        <th>Covering index</th>
                    </tr>
                </thead>
                <tbody>
                    {pageDuplicates.map((d, i) => (
                        <tr key={start + i}>
                            <td className="index-candidates-table__table">{d.table}</td>
                            <td title={d.redundantColumns.join(', ')}>{d.redundantIndex}</td>
                            <td title={d.coveringColumns.join(', ')}>{d.coveringIndex}</td>
                        </tr>
                    ))}
                </tbody>
            </table>

            <ul className="index-candidates-table__rationales">
                {pageDuplicates.map((d, i) => (
                    <li key={start + i}>
                        <strong>{d.redundantIndex}:</strong> {d.explanation}
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