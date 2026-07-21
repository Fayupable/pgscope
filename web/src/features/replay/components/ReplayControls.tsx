import { useRef } from 'react'
import type { useSnapshotPlayer } from '../hooks/useSnapshotPlayer'
import './ReplayControls.css'

interface ReplayControlsProps {
    player: ReturnType<typeof useSnapshotPlayer>
}

export function ReplayControls({ player }: ReplayControlsProps) {
    const fileInputRef = useRef<HTMLInputElement>(null)

    function handleFileSelected(event: React.ChangeEvent<HTMLInputElement>) {
        const file = event.target.files?.[0]
        if (file) player.load(file)
    }

    return (
        <div className="replay-controls">
            <button className="replay-controls__load" onClick={() => fileInputRef.current?.click()}>
                Load JSON
            </button>
            <input
                ref={fileInputRef}
                type="file"
                accept="application/json"
                onChange={handleFileSelected}
                hidden
            />

            {player.totalCount > 0 && (
                <>
                    <button className="replay-controls__play" onClick={player.isPlaying ? player.pause : player.play}>
                        {player.isPlaying ? '⏸' : '▶'}
                    </button>

                    <input
                        className="replay-controls__scrubber"
                        type="range"
                        min={0}
                        max={player.totalCount - 1}
                        value={player.currentIndex}
                        onChange={(e) => player.seek(Number(e.target.value))}
                    />

                    <span className="replay-controls__position">
                        {player.currentIndex + 1} / {player.totalCount}
                    </span>

                    <select value={player.speed} onChange={(e) => player.setSpeed(Number(e.target.value))}>
                        <option value={0.5}>0.5x</option>
                        <option value={1}>1x</option>
                        <option value={2}>2x</option>
                        <option value={4}>4x</option>
                    </select>
                </>
            )}

            {player.error && <p className="replay-controls__error">{player.error}</p>}
        </div>
    )
}
