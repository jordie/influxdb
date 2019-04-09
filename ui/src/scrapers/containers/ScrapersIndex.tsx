// Libraries
import React, {Component} from 'react'
import {withRouter, WithRouterProps} from 'react-router'
import {connect} from 'react-redux'

// Components
import {Page} from 'src/pageLayout'
import {Tabs} from 'src/clockface'
import TabbedPageSection from 'src/shared/components/tabbed_page/TabbedPageSection'
import OrgHeader from 'src/organizations/containers/OrgHeader'
import OrganizationNavigation from 'src/organizations/components/OrganizationNavigation'
import GetResources, {
  ResourceTypes,
} from 'src/configuration/components/GetResources'
import Scrapers from 'src/scrapers/components/Scrapers'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'

// Actions
import * as NotificationsActions from 'src/types/actions/notifications'
import * as notifyActions from 'src/shared/actions/notifications'

// Types
import {Organization} from '@influxdata/influx'
import {AppState} from 'src/types'

interface DispatchProps {
  notify: NotificationsActions.PublishNotificationActionCreator
}

interface StateProps {
  org: Organization
}

type Props = WithRouterProps & DispatchProps & StateProps

@ErrorHandling
class ScrapersIndex extends Component<Props> {
  public render() {
    const {org, notify} = this.props

    return (
      <>
        <Page titleTag={org.name}>
          <OrgHeader />
          <Page.Contents fullWidth={false} scrollable={true}>
            <div className="col-xs-12">
              <Tabs>
                <OrganizationNavigation tab="scrapers" orgID={org.id} />
                <Tabs.TabContents>
                  <TabbedPageSection
                    id="org-view-tab--scrapers"
                    url="scrapers"
                    title="Scrapers"
                  >
                    <GetResources resource={ResourceTypes.Scrapers}>
                      <GetResources resource={ResourceTypes.Buckets}>
                        <Scrapers orgName={org.name} notify={notify} />
                      </GetResources>
                    </GetResources>
                  </TabbedPageSection>
                </Tabs.TabContents>
              </Tabs>
            </div>
          </Page.Contents>
        </Page>
        {this.props.children}
      </>
    )
  }
}

const mstp = (state: AppState, props: Props) => {
  const {
    orgs: {items},
  } = state
  const org = items.find(o => o.id === props.params.orgID)
  return {
    org,
  }
}

const mdtp: DispatchProps = {
  notify: notifyActions.notify,
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(withRouter<{}>(ScrapersIndex))
