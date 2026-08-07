'use client';

import { memo } from 'react';
import type { Entity, EntityKind } from '../types';

const kindLabels: Record<EntityKind, string> = {
  organization: 'Organization',
  location: 'Location',
  person: 'Person',
  date: 'Date',
};

// MainClaimSection presents the claim under review plus the entities and
// keywords extracted from the input, so the user immediately sees what is
// actually being asserted.
function MainClaimSection({
  claim,
  entities,
  keywords,
}: {
  claim?: string;
  entities?: Entity[];
  keywords?: string[];
}) {
  if (!claim && !entities?.length && !keywords?.length) {
    return null;
  }

  const entityList = entities ?? [];
  const keywordList = keywords ?? [];

  return (
    <section
      aria-label="Main claim"
      className="rounded-xl border border-slate-200 bg-white p-4 shadow dark:border-slate-800 dark:bg-slate-900"
    >
      <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Main Claim</h3>
      {claim && (
        <p className="mt-2 text-sm leading-relaxed text-slate-700 dark:text-slate-300">{claim}</p>
      )}

      {(entityList.length || keywordList.length) && (
        <div className="mt-3 flex flex-wrap items-start gap-x-6 gap-y-3 text-sm">
          {entityList.length > 0 && (
            <div>
              <h4 className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Entities
              </h4>
              <ul className="mt-1 flex flex-wrap gap-1.5">
                {entityList.map((entity) => (
                  <li
                    key={`${entity.kind}-${entity.name}`}
                    className="rounded-md bg-cyan-50 px-2 py-1 text-xs font-medium text-cyan-800 dark:bg-cyan-950/40 dark:text-cyan-300"
                    title={kindLabels[entity.kind] ?? entity.kind}
                  >
                    {entity.name}
                  </li>
                ))}
              </ul>
            </div>
          )}
          {keywordList.length > 0 && (
            <div>
              <h4 className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
                Keywords
              </h4>
              <ul className="mt-1 flex flex-wrap gap-1.5">
                {keywordList.map((keyword) => (
                  <li
                    key={keyword}
                    className="rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-300"
                  >
                    {keyword}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </section>
  );
}

export default memo(MainClaimSection);
