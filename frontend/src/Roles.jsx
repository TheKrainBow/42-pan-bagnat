import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
} from '@tanstack/react-table';
import './Roles.css';

const Roles = () => {
  const [filterQuery, setFilterQuery] = useState('');
  const [debouncedFilter, setDebouncedFilter] = useState('');
  const [roles, setRoles] = useState([]);
  const [nextPage, setNextPage] = useState('');
  const [orderQuery, setOrderQuery] = useState('name');
  const [isLoading, setIsLoading] = useState(false);
  const loadingRef = useRef(false);
  const scrollRef = useRef(null);
  const isFirst = useRef(true);

  // fetch roles (with optional append for infinite‐scroll)
  const fetchRoles = useCallback(async (append = false, token = '') => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setIsLoading(true);

    try {
      const params = new URLSearchParams();
      params.set('order', orderQuery);
      if (debouncedFilter) params.set('filter', debouncedFilter);
      if (token) {
        params.set('next_page_token', token);
      } else {
        params.set('limit', 20);
      }

      const res = await fetch(`http://localhost:8080/api/v1/roles?${params.toString()}`);
      const data = await res.json();

      setRoles(prev =>
        append
          ? [...prev, ...data.roles]
          : Array.isArray(data.roles) ? data.roles : []
      );
      setNextPage(data.next_page_token);
    } catch (err) {
      console.error(err);
      if (!append) setRoles([]);
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
    }
  }, [orderQuery, debouncedFilter]);

  // infinite‐scroll handler
  const onScroll = useCallback(() => {
    const el = scrollContainerRef.current;
    if (!el || isLoading || !nextPage) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 10) {
      fetchRoles(true, nextPage);
    }
  }, [fetchRoles, isLoading, nextPage]);

  // attach scroll listener
  useEffect(() => {
    const el = scrollRef.current;
    el?.addEventListener('scroll', onScroll);
    return () => el?.removeEventListener('scroll', onScroll);
  }, [onScroll]);

  // debounce filter
  useEffect(() => {
    const t = setTimeout(() => setDebouncedFilter(filterQuery), 300);
    return () => clearTimeout(t);
  }, [filterQuery]);

  // initial load & on order/filter change
  useEffect(() => {
    if (isFirst.current) {
      isFirst.current = false;
      fetchRoles();
    } else {
      // if user changes sort or filter, reload from scratch
      fetchRoles(false, '');
    }
  }, [fetchRoles]);

  // table columns
  const columns = useMemo(() => [
    {
      header: 'Role',
      accessorKey: 'name',
      cell: info => (
        <span
          className="role-badge"
          style={{ backgroundColor: `#${info.row.original.color.replace('0x','')}` }}
        >
          {info.getValue()}
        </span>
      ),
    },
    {
      header: 'Members',
      accessorKey: 'members_count',
      disableSort: true,
      cell: info => (
        <><img src="/icons/user.svg" alt="members" className="icon-small" /> {info.getValue()}</>
      ),
    },
    {
      header: 'Applications',
      accessorKey: 'applications',
      disableSort: true,
      cell: info => (
        <div className="apps-cell">
          {info.getValue().map(app => (
            <img
              key={app.id}
              src={app.icon_url}
              alt={app.name}
              title={app.name}
              className="app-icon"
            />
          ))}
        </div>
      ),
    },
  ], []);

  // sort arrows
  const getSortDir = colId => {
    if (orderQuery === colId) return 'asc';
    if (orderQuery === `-${colId}`) return 'desc';
    return '';
  };
  const toggleSort = colId => {
    const isDesc = orderQuery === `-${colId}`;
    const newOrder = isDesc ? colId : `-${colId}`;
    setOrderQuery(newOrder === orderQuery ? '' : newOrder);
  };

  const table = useReactTable({
    data: roles,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  return (
    <div className="p-4">
      <div className="roles-header">
        <h2>Roles</h2>
        <div className="roles-controls">
          <div className="search-wrap">
            <img src="/icons/search.png" alt="Search" className="search-icon" />
            <input
              type="text"
              className="search-input"
              placeholder="Search roles..."
              value={filterQuery}
              onChange={e => setFilterQuery(e.target.value)}
            />
          </div>
          <button className="filter-btn">
            <img src="/icons/filter.svg" alt="Filter" />
          </button>
          <button className="add-btn" onClick={() => {/* open add-role modal */}}>
            <img src="/icons/plus.svg" alt="Add Role" />
          </button>
        </div>
      </div>

      <div className="roles-table-container" ref={scrollRef}>
        <table className="roles-table">
          <thead>
            {table.getHeaderGroups().map(hg => (
              <tr key={hg.id}>
                {hg.headers.map(header => {
                  const colId = header.id;
                  const sortable = !header.column.columnDef.disableSort;
                  const dir = getSortDir(colId);

                  return (
                    <th
                      key={colId}
                      className={sortable ? 'sortable' : ''}
                      onClick={sortable ? () => toggleSort(colId) : undefined}
                    >
                      <div className="th-content">
                        {flexRender(header.column.columnDef.header, header.getContext())}
                        {sortable && (
                          <span className="arrows">
                            <span style={{opacity: dir==='asc'?1:0.3}}>▲</span>
                            <span style={{opacity: dir==='desc'?1:0.3}}>▼</span>
                          </span>
                        )}
                      </div>
                    </th>
                  );
                })}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.map(row => (
              <tr key={row.id}>
                {row.getVisibleCells().map(cell => (
                  <td key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>

        {nextPage && !isLoading && (
          <div className="load-more">
            <button onClick={() => fetchRoles(true, nextPage)}>Load More</button>
          </div>
        )}
        {isLoading && <div className="loading">Loading…</div>}
      </div>
    </div>
  );
};

export default Roles;
