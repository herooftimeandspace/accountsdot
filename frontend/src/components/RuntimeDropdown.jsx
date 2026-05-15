import { useEffect, useId, useMemo, useRef, useState } from "react";

function normalizedOption(option) {
  if (typeof option === "string" || typeof option === "number") {
    return { value: String(option), label: String(option) };
  }
  const value = option.value ?? option.id ?? option.label ?? "";
  return {
    ...option,
    value: String(value),
    label: String(option.label ?? value),
  };
}

function optionId(listboxId, value) {
  return `${listboxId}-${String(value).replace(/[^a-zA-Z0-9_-]+/g, "-")}`;
}

/**
 * RuntimeSelectDropdown renders the shared implemented-page select primitive.
 * It owns the visible control and listbox overlay so `.pen` text, drawer
 * content, and browser-native menus cannot drift behind the active dropdown.
 */
export function RuntimeSelectDropdown({
  label,
  value,
  options,
  onChange,
  className = "",
  buttonClassName = "",
  listClassName = "",
}) {
  const generatedId = useId();
  const rootRef = useRef(null);
  const [isOpen, setIsOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const normalizedOptions = useMemo(() => options.map(normalizedOption), [options]);
  const selectedIndex = Math.max(0, normalizedOptions.findIndex((option) => option.value === String(value)));
  const selectedOption = normalizedOptions[selectedIndex] ?? normalizedOptions[0] ?? { value: "", label: "" };
  const listboxId = `${generatedId}-listbox`;

  useEffect(() => {
    if (!isOpen) {
      return undefined;
    }
    setActiveIndex(selectedIndex);
    const handlePointerDown = (event) => {
      if (!rootRef.current?.contains(event.target)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [isOpen, selectedIndex]);

  function selectOption(option) {
    if (!option) {
      return;
    }
    onChange?.(option.value);
    setIsOpen(false);
  }

  function handleKeyDown(event) {
    if (event.key === "Escape") {
      setIsOpen(false);
      return;
    }
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      if (!normalizedOptions.length) {
        return;
      }
      event.preventDefault();
      setIsOpen(true);
      setActiveIndex((current) => {
        const direction = event.key === "ArrowDown" ? 1 : -1;
        return (current + direction + normalizedOptions.length) % normalizedOptions.length;
      });
      return;
    }
    if ((event.key === "Enter" || event.key === " ") && isOpen) {
      event.preventDefault();
      selectOption(normalizedOptions[activeIndex]);
    }
  }

  return (
    <div ref={rootRef} className={`runtime-dropdown ${className}`.trim()} onKeyDown={handleKeyDown}>
      <button
        type="button"
        className={`runtime-dropdown__button ${buttonClassName}`.trim()}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-label={label}
        aria-controls={listboxId}
        aria-activedescendant={isOpen ? optionId(listboxId, normalizedOptions[activeIndex]?.value ?? "") : undefined}
        onClick={() => setIsOpen((current) => !current)}
      >
        <span>{selectedOption.label}</span>
        <span className="runtime-dropdown__chevron" aria-hidden="true" />
      </button>
      {isOpen ? (
        <div className={`runtime-dropdown__list ${listClassName}`.trim()} id={listboxId} role="listbox" aria-label={label}>
          {normalizedOptions.map((option, index) => (
            <button
              key={option.value}
              id={optionId(listboxId, option.value)}
              type="button"
              role="option"
              aria-selected={option.value === selectedOption.value}
              className={`runtime-dropdown__option ${index === activeIndex ? "runtime-dropdown__option--active" : ""}`.trim()}
              onMouseEnter={() => setActiveIndex(index)}
              onClick={() => selectOption(option)}
            >
              {option.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}

/**
 * RuntimeCombobox renders an input-anchored listbox for DEV autocomplete
 * flows. Callers keep owning filtering and commit semantics while this shared
 * primitive handles overlay placement, stacking, Escape, and pointer selection.
 */
export function RuntimeCombobox({
  label,
  value,
  options,
  onInput,
  onCommit,
  placeholder = "Search",
  inputId,
  className = "",
}) {
  const generatedId = useId();
  const rootRef = useRef(null);
  const [isOpen, setIsOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const normalizedOptions = useMemo(() => options.map(normalizedOption), [options]);
  const listboxId = `${generatedId}-listbox`;
  const activeOption = normalizedOptions[activeIndex];

  useEffect(() => {
    if (!isOpen) {
      return undefined;
    }
    const handlePointerDown = (event) => {
      if (!rootRef.current?.contains(event.target)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [isOpen]);

  useEffect(() => {
    setActiveIndex(0);
  }, [value, normalizedOptions.length]);

  function commit(optionValue) {
    onCommit?.(optionValue);
    setIsOpen(false);
  }

  function handleKeyDown(event) {
    if (event.key === "Escape") {
      setIsOpen(false);
      return;
    }
    if ((event.key === "ArrowDown" || event.key === "ArrowUp") && normalizedOptions.length) {
      event.preventDefault();
      setIsOpen(true);
      setActiveIndex((current) => {
        const direction = event.key === "ArrowDown" ? 1 : -1;
        return (current + direction + normalizedOptions.length) % normalizedOptions.length;
      });
      return;
    }
    if (event.key === "Enter" && isOpen && activeOption) {
      event.preventDefault();
      commit(activeOption.value);
    }
  }

  return (
    <div ref={rootRef} className={`runtime-combobox ${className}`.trim()}>
      <input
        id={inputId}
        type="search"
        role="combobox"
        aria-label={label}
        aria-controls={listboxId}
        aria-expanded={isOpen && normalizedOptions.length > 0}
        aria-autocomplete="list"
        aria-activedescendant={isOpen && activeOption ? optionId(listboxId, activeOption.value) : undefined}
        value={value}
        placeholder={placeholder}
        onFocus={() => setIsOpen(true)}
        onChange={(event) => {
          onInput?.(event.target.value);
          setIsOpen(true);
        }}
        onBlur={(event) => onCommit?.(event.target.value)}
        onKeyDown={handleKeyDown}
      />
      {isOpen && normalizedOptions.length ? (
        <div className="runtime-combobox__list" id={listboxId} role="listbox" aria-label={`${label} results`}>
          {normalizedOptions.map((option, index) => (
            <button
              key={option.value}
              id={optionId(listboxId, option.value)}
              type="button"
              role="option"
              aria-selected={index === activeIndex}
              className={`runtime-combobox__option ${index === activeIndex ? "runtime-combobox__option--active" : ""}`.trim()}
              onMouseDown={(event) => event.preventDefault()}
              onMouseEnter={() => setActiveIndex(index)}
              onClick={() => commit(option.value)}
            >
              {option.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
