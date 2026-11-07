"use client";

export interface TabItem {
  id: string;
  label: string;
}

export function Tabs({
  tabs,
  value,
  onChange,
  label,
}: {
  tabs: readonly TabItem[];
  value: string;
  onChange: (id: string) => void;
  label: string;
}) {
  return (
    <div className="ui-tabs" role="tablist" aria-label={label}>
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          className="ui-tab"
          aria-selected={tab.id === value}
          tabIndex={tab.id === value ? 0 : -1}
          onClick={() => onChange(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}
