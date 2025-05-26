import React, { useMemo, useEffect, useState, useRef, useCallback } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
} from '@tanstack/react-table';
import './User.css';
import { Header } from '../components/Header';
import { RoleBadge } from '../components/RoleBadge';
import { ArrayHeader } from '../components/ArrayHeader';

const User = () => {
  const [filterQuery, setFilterQuery] = useState('');
  const [debouncedFilter, setDebouncedFilter] = useState('');
  const [users, setUsers] = useState([]);
  const [nextPage, setNextPage] = useState('');
  const [orderQuery, setOrderQuery] = useState('-last_seen');
  const loadingRef = useRef(false);
  const [isLoading, setIsLoading] = useState(false);
  const scrollContainerRef = useRef(null);

  const fetchUsers = useCallback(async (append = false, token = '') => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setIsLoading(true); // Start loading

    try {
      const params = new URLSearchParams();
      params.set('order', orderQuery);
      if (debouncedFilter !== '') {
        params.set('filter', debouncedFilter);
      }
      if (token) {
        params.set('next_page_token', token);
      } else {
        params.set('limit', 20); // Keep this for first load
      }

      const response = await fetch(`http://localhost:8080/api/v1/users?${params.toString()}`);
      const data = await response.json();

      setUsers(prev =>
        append ? [...prev, ...data.users]
        : (Array.isArray(data.users) ? data.users : [])
      );
      setNextPage(data.next_page_token);
    } catch (error) {
      console.error('Error fetching users:', error);
      if (!append) setUsers([]);
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
    }
  }, [orderQuery, debouncedFilter]);


  // Function to handle column sorting
  const handleSort = (column) => {
    const isDesc = orderQuery.startsWith(`-${column}`);
    const newOrder = isDesc ? column : `-${column}`;
    setOrderQuery(newOrder === orderQuery ? '' : newOrder); // toggle order
  };
  
  const handleScroll = useCallback(() => {
    const el = scrollContainerRef.current;
    if (!el || isLoading || !nextPage) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 10) {
      fetchUsers(true, nextPage);
    }
  }, [nextPage, isLoading, fetchUsers]);
  
  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    scrollContainer?.addEventListener('scroll', handleScroll);  
    return () => scrollContainer?.removeEventListener('scroll', handleScroll);
  }, [handleScroll]);
  
  useEffect(() => {
    const timeout = setTimeout(() => {
      setDebouncedFilter(filterQuery);
    }, 300);
    return () => clearTimeout(timeout);
  }, [filterQuery]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const columns = useMemo(
    () => [
      {
        header: 'Picture',
        disableSort: true,
        cell: info => (
          <img
            src={info.row.original.ft_photo}
            alt={info.row.original.ft_login}
            className="user-picture"
          />
        ),
      },
      {
        header: 'Login',
        accessorKey: 'ft_login',
        cell: info => {
          const isStaff = info.row.original.is_staff;
          const login = info.getValue();
          return (
            <span className={isStaff ? 'staff-login' : ''}>
              {login}
            </span>
          );
        }
      },
      { header: '42 ID', accessorKey: 'ft_id' },
      {
        header: 'Roles',
        accessorKey: 'roles',
        disableSort: true,
        cell: info =>
          info.getValue().map(role => (
              <RoleBadge key={role.id} hexColor={role.color}>
                {role.name}
              </RoleBadge>
          )),
      },
      {
        header: 'Last Seen',
        accessorKey: 'last_seen',
        cell: info =>
          new Date(info.getValue()).toLocaleString('fr-FR'),
      },
    ],
    []
  );

  const getSortDirection = (columnId) => {
    if (orderQuery === columnId) return 'asc';
    if (orderQuery === `-${columnId}`) return 'desc';
    return '';
  };

  const table = useReactTable({
    data: users,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });
  return (
    <div className="p-4">
      <Header
        title="Users"
        value={filterQuery}
        onChange={(e) => setFilterQuery(e.target.value)}
      />

      <div className="user-table-container" ref={scrollContainerRef}>
        <table className="user-table">
          <ArrayHeader
            table={table}
            handleSort={handleSort}
            getSortDirection={getSortDirection}
          />
          <tbody>
            {table.getRowModel().rows.map(row => (
              <tr key={row.id} className="user-row">
                {row.getVisibleCells().map(cell => {
                  return (
                    <td key={cell.id} className={cell.column.columnDef.header === 'Picture' ? 'user-small-column' : ''}>
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
    </div>
  );
};

export default User;
