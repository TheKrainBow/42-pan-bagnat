import React from 'react';
import { flexRender } from '@tanstack/react-table'; // adjust if you import it elsewhere
import './ArrayHeader.css'

export const ArrayHeader = ({ table, handleSort, getSortDirection }) => {
  return (
    <thead className="array-header">
      {table.getHeaderGroups().map(headerGroup => (
        <tr key={headerGroup.id}>
          {headerGroup.headers.map(header => {
            const isSortable = !header.column.columnDef.disableSort;
            const sortDir = getSortDirection(header.id);

            return (
              <th
                key={header.id}
                onClick={isSortable ? () => handleSort(header.id) : undefined}
                className={`array-header-cell ${isSortable ? 'sortable' : 'disabled-sort'}`}
              >
                <div className={`array-header-content ${isSortable ? 'sortable' : 'disabled-sort'}`}>
                  {flexRender(header.column.columnDef.header, header.getContext())}
                  {isSortable && (
                    <span className="sort-arrows">
                      <span style={{ opacity: sortDir === 'asc' ? 1 : 0.5 }}>▲</span>
                      <span style={{ opacity: sortDir === 'desc' ? 1 : 0.5 }}>▼</span>
                    </span>
                  )}
                </div>
              </th>
            );
          })}
        </tr>
      ))}
    </thead>
  );
};

export default ArrayHeader;
