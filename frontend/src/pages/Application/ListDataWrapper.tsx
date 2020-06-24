import React from "react";
import { RootState } from "../../reducers";
import { connect } from "react-redux";
import { ThunkDispatch } from "redux-thunk";
import { Actions } from "../../types";
import { loadApplicationsAction } from "../../actions/application";

const mapStateToProps = (state: RootState) => {
  const applicationsState = state.get("applications");
  const applications = applicationsState.get("applications");

  return {
    applications,
    activeNamespaceName: state.get("namespaces").get("active"),
    isLoading: applicationsState.get("isListLoading"),
    isFirstLoaded: applicationsState.get("isListFirstLoaded"),
  };
};

export interface WithApplicationsListDataProps extends ReturnType<typeof mapStateToProps> {
  dispatch: ThunkDispatch<RootState, undefined, Actions>;
}

export const ApplicationListDataWrapper = (WrappedComponent: React.ComponentType<any>) => {
  const WithdApplicationsData: React.ComponentType<WithApplicationsListDataProps> = class extends React.Component<
    WithApplicationsListDataProps
  > {
    private interval?: number;

    private loadData = () => {
      this.props.dispatch(loadApplicationsAction());
      // Just for refresh mestrics. Reload per minute,
      this.interval = window.setTimeout(this.loadData, 60000);
    };

    componentDidMount() {
      this.loadData();
    }

    componentWillUnmount() {
      if (this.interval) {
        window.clearTimeout(this.interval);
      }
    }

    render() {
      return <WrappedComponent {...this.props} />;
    }
  };

  WithdApplicationsData.displayName = `WithdApplicationsData(${getDisplayName(WrappedComponent)})`;

  return connect(mapStateToProps)(WithdApplicationsData);
};

function getDisplayName(WrappedComponent: React.ComponentType) {
  return WrappedComponent.displayName || WrappedComponent.name || "Component";
}
