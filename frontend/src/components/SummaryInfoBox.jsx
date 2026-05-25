/**
 * SummaryInfoBox is the shared runtime primitive for actionable metric cards.
 * Implemented pages use it when a large numeric value should drive an operator
 * to a concrete filter, navigation target, drawer, or decision path instead of
 * sitting on the page as passive decoration.
 */
export function SummaryInfoBox({
  label,
  value,
  helper,
  tone = "neutral",
  actionLabel,
  onAction,
  href,
  active = false,
  className = "",
  style,
}) {
  const classes = [
    "summary-info-box",
    `summary-info-box--${tone}`,
    active ? "summary-info-box--active" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");
  const accessibleLabel = actionLabel
    ? `${label}: ${value}. ${actionLabel}`
    : `${label}: ${value}`;
  const content = (
    <>
      <span className="summary-info-box__label">{label}</span>
      <span className="summary-info-box__value">{value}</span>
      {helper ? <span className="summary-info-box__helper">{helper}</span> : null}
    </>
  );

  if (href) {
    return (
      <a className={classes} href={href} aria-label={accessibleLabel} style={style}>
        {content}
      </a>
    );
  }
  if (onAction) {
    return (
      <button
        type="button"
        className={classes}
        aria-label={accessibleLabel}
        aria-pressed={active ? "true" : "false"}
        onClick={onAction}
        style={style}
      >
        {content}
      </button>
    );
  }
  return (
    <article className={classes} aria-label={accessibleLabel} style={style}>
      {content}
    </article>
  );
}

/**
 * SummaryInfoBoxGrid keeps repeated metric boxes aligned while each page owns
 * the actual metric-to-action mapping and any .pen-derived absolute placement.
 */
export function SummaryInfoBoxGrid({ children, className = "", style }) {
  return (
    <section className={`summary-info-box-grid ${className}`.trim()} aria-label="Summary actions" style={style}>
      {children}
    </section>
  );
}
