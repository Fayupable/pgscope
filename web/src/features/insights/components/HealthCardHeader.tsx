// Shared header for every health-card: status icon + title were previously
// copy-pasted identically into all nine card components. Pulling it out
// here means the always-visible plain-language `subtitle` — what this card
// actually checks, independent of whether it currently has anything to
// report — only has to be written once per card's data, not once per
// card's markup.
export function HealthCardHeader({ title, subtitle, ok }: { title: string; subtitle: string; ok: boolean }) {
    return (
        <div className="health-card__header">
            <div className="health-card__header-text">
                <div className="health-card__title-row">
                    <span className={'health-card__status' + (ok ? ' health-card__status--ok' : ' health-card__status--warning')}>
                        {ok ? '✓' : '⚠'}
                    </span>
                    <span className="health-card__title">{title}</span>
                </div>
                <p className="health-card__subtitle">{subtitle}</p>
            </div>
        </div>
    )
}
