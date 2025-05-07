import React, { useMemo, useEffect, useState, useRef, useCallback } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
} from '@tanstack/react-table';
import './User.css';

const User = () => {
  const [filterQuery, setFilterQuery] = useState('');
  const [debouncedFilter, setDebouncedFilter] = useState('');
  const [users, setUsers] = useState([]);
  const [nextPage, setNextPage] = useState('');
  const [orderQuery, setOrderQuery] = useState('-last_seen');
  const loadingRef = useRef(false);
  const [isLoading, setIsLoading] = useState(false);
  const scrollContainerRef = useRef(null);

  const isFirstRender = useRef(true);
  

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

      setUsers(prev => append ? [...prev, ...data.users] : (Array.isArray(data.users) ? data.users : []));
      setNextPage(data.next_page_token);
    } catch (error) {
      console.error('Error fetching users:', error);
      if (!append) setUsers([]);
    } finally {
      loadingRef.current = false;
      setIsLoading(false); // End loading
    }
  }, [orderQuery, debouncedFilter]);


  // Function to handle column sorting
  const handleSort = (column) => {
    const isDesc = orderQuery.startsWith(`-${column}`);
    const newOrder = isDesc ? column : `-${column}`;
    setOrderQuery(newOrder === orderQuery ? '' : newOrder); // toggle order
  };
  
  const handleScroll = useCallback(() => {
    const scrollTop = scrollContainerRef.current.scrollTop;
    const scrollHeight = scrollContainerRef.current.scrollHeight;
    const clientHeight = scrollContainerRef.current.clientHeight;
  
    // Check if we are near the bottom (with a small buffer of 10px)
    if (scrollTop + clientHeight >= scrollHeight - 10 && nextPage && !isLoading) {
      fetchUsers(true, nextPage);
    }
  }, [nextPage, isLoading, fetchUsers]);
  
  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll);
    }
  
    return () => {
      if (scrollContainer) {
        scrollContainer.removeEventListener('scroll', handleScroll);
      }
    };
  }, [handleScroll]);
  
  // Prevent calling scroll handler too frequently (debounce-like behavior)
  useEffect(() => {
    const timeout = setTimeout(() => {
      setDebouncedFilter(filterQuery);
    }, 300);  // Adjust debounce delay as needed
    return () => clearTimeout(timeout);
  }, [filterQuery]);

  useEffect(() => {
    const timeout = setTimeout(() => setDebouncedFilter(filterQuery), 200);
    return () => clearTimeout(timeout);
  }, [filterQuery]);

  useEffect(() => {
    if (isFirstRender.current === false) {
      isFirstRender.current = false;
      return;
    }
    fetchUsers();
  }, [fetchUsers]);


  useEffect(() => {
    const timeout = setTimeout(() => {
      setDebouncedFilter(filterQuery);
    }, 300);
    return () => clearTimeout(timeout);
  }, [filterQuery]);

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
            <span
              key={role.id}
              className="user-role"
              style={{ backgroundColor: `#${role.color.replace('0x', '')}` }}
            >
              {role.name}
            </span>
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
      <div className="user-header-bar">
        <h2>User List</h2>
        <div className="user-search-container">
          <img src="/icons/search.png" alt="Search" className="search-icon-inside" />
          <input
            className="user-search with-icon"
            type="text"
            placeholder="Search..."
            value={filterQuery}
            onChange={(e) => setFilterQuery(e.target.value)}
          />
        </div>
      </div>

      <div className="user-table-container" ref={scrollContainerRef}>
        <table className="user-table">
          <thead className="user-header">
            {table.getHeaderGroups().map(headerGroup => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map(header => {
                  const isSortable = !header.column.columnDef.disableSort;
                  const sortDir = getSortDirection(header.id);

                  return (
                    <th
                      key={header.id}
                      onClick={isSortable ? () => handleSort(header.id) : undefined}
                      className={`user-cell ${isSortable ? 'sortable' : 'disabled-sort'} ${(header.column.columnDef.header === 'Picture') ? 'small-column' : ''}`}
                    >
                      <div className={`header-content ${isSortable ? 'sortable' : 'disabled-sort'}`}>
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
          <tbody>
            {table.getRowModel().rows.map(row => (
              <tr key={row.id} className="user-row">
                {row.getVisibleCells().map(cell => {
                  return (
                    <td key={cell.id} className={cell.column.columnDef.header === 'Picture' ? 'small-column' : ''}>
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
            <button className="load-more-button" onClick={() => fetchUsers(true, nextPage)}>
              Load More
            </button>
          </div>
        )}
        {isLoading && (
          <div className="loading-icon">Loading...</div>
        )}
      </div>
    </div>
  );
};

export default User;
