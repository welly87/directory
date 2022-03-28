import Certificate from 'modules/dashboard/Certificate/Certificate';
import React, { Suspense } from 'react';
import { Routes, Route, Link } from 'react-router-dom';

const Home = React.lazy(() => import('modules/home'));
const StartPage = React.lazy(() => import('modules/start'));

const AppRouter: React.FC = () => {
  return (
    <Suspense fallback="loading...">
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/start" element={<StartPage />} />
        <Route path="/certificate" element={<Certificate />} />
      </Routes>
    </Suspense>
  );
};

export default AppRouter;
