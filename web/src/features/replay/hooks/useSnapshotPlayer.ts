import { useEffect, useState } from 'react'
import type { Snapshot } from '../../../shared/types/snapshot'
import { parseSnapshotFile } from '../../../shared/utils/parseSnapshotFile'

export function useSnapshotPlayer() {
    const [snapshots, setSnapshots] = useState<Snapshot[]>([])
    const [currentIndex, setCurrentIndex] = useState(0)
    const [isPlaying, setIsPlaying] = useState(false)
    const [speed, setSpeed] = useState(1)
    const [error, setError] = useState<string | null>(null)

    async function load(file: File) {
        try {
            setError(null)
            const parsed = await parseSnapshotFile(file)
            setSnapshots(parsed)
            setCurrentIndex(0)
            setIsPlaying(false)
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load file')
        }
    }

    function play() {
        if (snapshots.length === 0) return
        if (currentIndex >= snapshots.length - 1) setCurrentIndex(0)
        setIsPlaying(true)
    }

    function pause() {
        setIsPlaying(false)
    }

    function seek(index: number) {
        const clamped = Math.max(0, Math.min(index, snapshots.length - 1))
        setCurrentIndex(clamped)
    }

    useEffect(() => {
        if (!isPlaying) return

        const intervalMs = 1000 / speed
        const timer = setInterval(() => {
            setCurrentIndex((prev) => {
                if (prev >= snapshots.length - 1) {
                    setIsPlaying(false)
                    return prev
                }
                return prev + 1
            })
        }, intervalMs)

        return () => clearInterval(timer)
    }, [isPlaying, speed, snapshots.length])

    return {
        snapshots,
        currentIndex,
        currentSnapshot: snapshots[currentIndex] ?? null,
        totalCount: snapshots.length,
        isPlaying,
        speed,
        error,
        load,
        play,
        pause,
        seek,
        setSpeed,
    }
}