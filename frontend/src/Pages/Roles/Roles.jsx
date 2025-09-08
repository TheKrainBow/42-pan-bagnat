import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
} from '@tanstack/react-table';
import { useNavigate } from 'react-router-dom'
import './Roles.css';
import AppIcon from 'ui/molecules/AppIcon/AppIcon';
import Header from 'Global/Header/Header';
import RoleBadge from 'ui/molecules/Badges/RoleBadge';
import ArrayHeader from 'ui/molecules/ArrayHeader/ArrayHeader';
import Button from 'ui/atoms/Button/Button';
import RoleImport from './RoleImport/RoleImport';
import { fetchWithAuth } from 'Global/utils/Auth';

const Roles = () => {
  const [filterQuery, setFilterQuery] = useState('');
  const [debouncedFilter, setDebouncedFilter] = useState('');
  const [roles, setRoles] = useState([]);
  const [nextPage, setNextPage] = useState('');
  const [orderQuery, setOrderQuery] = useState('name');
  const [isLoading, setIsLoading] = useState(false);
  const loadingRef = useRef(false);
  const scrollContainerRef = useRef(null);
  const isFirst = useRef(true);
  const [showRoleImport, setShowRoleImport] = useState(false);
  const navigate = useNavigate();

  const handleCreateSuccess = () => {
    setRoles([]);
    setNextPage('');
    fetchRoles(false, '');
  };
  // fetch roles (with optional append for infinite‐scroll)
  const fetchRoles = useCallback(async (append = false, token = '') => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setIsLoading(true);

    try {
      const params = new URLSearchParams();
      params.set('order', orderQuery);
      if (debouncedFilter !== '') {
        params.set('filter', debouncedFilter);
      }
      if (token) {
        params.set('next_page_token', token);
      } else {
        params.set('limit', 20);
      }

      const res = await fetchWithAuth(`/api/v1/admin/roles?${params.toString()}`);
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


  const handleSort = (column) => {
    const isDesc = orderQuery.startsWith(`-${column}`);
    const newOrder = isDesc ? column : `-${column}`;
    setOrderQuery(newOrder === orderQuery ? '' : newOrder); // toggle order
  };

  
  const handleScroll = useCallback(() => {
    const el = scrollContainerRef.current;
    if (!el || isLoading || !nextPage) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 10) {
      fetchRoles(true, nextPage);
    }
  }, [nextPage, isLoading, fetchRoles]);

  // attach scroll listener
  useEffect(() => {
    const el = scrollContainerRef.current;
    el?.addEventListener('scroll', handleScroll);
    return () => el?.removeEventListener('scroll', handleScroll);
  }, [handleScroll]);

  // debounce filter
  useEffect(() => {
    const t = setTimeout(() => setDebouncedFilter(filterQuery), 300);
    return () => clearTimeout(t);
  }, [filterQuery]);

  useEffect(() => {
    fetchRoles();
  }, [fetchRoles]);

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
      header: 'Members',
      accessorFn: row => Array.isArray(row.users) ? row.users.length : 0,
      id: 'users',
      disableSort: true,
      cell: info => {
        const count = info.getValue();
        const classNamee = count === 0 ? "icon-small empty" : "icon-small";
        return (
          <div className="members-cell">
            <img src="/icons/users.png" alt="users" className={classNamee} />
            <span className="members-count">{count}</span>
          </div>
        );
      },
    },
    {
      header: 'Role',
      accessorKey: 'name',
      cell: info => (
        <RoleBadge role={info.row.original} onClick={() => navigate(`/admin/roles/${info.row.original.id}`)}>
          {info.getValue()}
        </RoleBadge>
      ),
    },
    {
      header: 'Modules',
      accessorKey: 'modules',
      disableSort: true,
      cell: info => {
        const modules = info.getValue();
        const fallback = '/icons/modules.png';
      
        return (
          <div className="role-apps-cell">
            {!Array.isArray(modules) || modules.length === 0 ? (
              <span style={{ opacity: 0.5 }}>No modules linked</span>
            ) : (
              modules.map(app => (
                <AppIcon key={app.id} app={app} fallback={fallback} />
              ))
            )}
          </div>
        );
      }
    },
  ], []);

  // sort arrows
  const getSortDirection = colId => {
    if (orderQuery === colId) return 'asc';
    if (orderQuery === `-${colId}`) return 'desc';
    return '';
  };

  const table = useReactTable({
    data: roles,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });
  return (
    <div className="p-4">
      <Header
        title="Roles"
        value={filterQuery}
        onChange={(e) => setFilterQuery(e.target.value)}
      />

      <div className="role-table-container" ref={scrollContainerRef}>
        <table className="role-table">
          <ArrayHeader
            table={table}
            handleSort={handleSort}
            getSortDirection={getSortDirection}
            headerRight={
              <Button
                label={'Add Role'}
                color="blue"
                onClick={() => setShowRoleImport(true)}
             />
            }
          />
          <tbody>
            {table.getRowModel().rows.map(row => (
              <tr
                key={row.id}
                className="role-row clickable-row"
                onClick={() => navigate(`/admin/roles/${row.original.id}`)}
              >
                {row.getVisibleCells().map(cell => {
                  return (
                  <td key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
        {nextPage && (
          <div className="load-more-wrapper">
            <div className="loader" style={{ width: 24, height: 24 }} />
          </div>
        )}
      </div>
      <RoleImport
        show={showRoleImport}
        onClose={() => setShowRoleImport(false)}
        onCreateSuccess={handleCreateSuccess}
      />
    </div>
  );
};

export default Roles;
