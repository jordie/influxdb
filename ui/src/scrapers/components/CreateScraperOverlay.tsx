// Libraries
import React, {PureComponent, ChangeEvent} from 'react'
import _ from 'lodash'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'

// Components
import {Form, Button} from '@influxdata/clockface'
import {Overlay} from 'src/clockface'
import CreateScraperForm from 'src/scrapers/components/CreateScraperForm'

// Actions
import {notify as notifyAction, notify} from 'src/shared/actions/notifications'
import {createScraper} from 'src/organizations/actions/orgView'

// Types
import {Bucket, ScraperTargetRequest} from '@influxdata/influx'
import {ComponentColor, ComponentStatus} from '@influxdata/clockface'
import {
  scraperCreateSuccess,
  scraperCreateFailed,
} from 'src/shared/copy/v2/notifications'
import {AppState} from 'src/types'

interface OwnProps {
  overrideBucketIDSelection?: string
  visible: boolean
}

interface StateProps {
  buckets: Bucket[]
}

interface DispatchProps {
  notify: typeof notifyAction
  onCreateScraper: typeof createScraper
}

type Props = OwnProps & StateProps & DispatchProps & WithRouterProps

interface State {
  scraper: ScraperTargetRequest
}

class CreateScraperOverlay extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props)

    const bucketID =
      this.props.overrideBucketIDSelection || this.props.buckets[0].id

    const orgID = this.props.buckets.find(b => b.id === bucketID).organizationID

    this.state = {
      scraper: {
        name: 'Name this Scraper',
        type: ScraperTargetRequest.TypeEnum.Prometheus,
        url: `${this.origin}/metrics`,
        orgID,
        bucketID,
      },
    }
  }

  componentDidUpdate(prevProps) {
    if (
      prevProps.visible === false &&
      this.props.visible === true &&
      this.props.overrideBucketIDSelection
    ) {
      const bucketID = this.props.overrideBucketIDSelection
      const orgID = this.props.buckets.find(b => b.id === bucketID)
        .organizationID

      const scraper = {
        ...this.state.scraper,
        bucketID,
        orgID,
      }
      this.setState({scraper})
    }
  }

  public render() {
    const {scraper} = this.state
    const {buckets} = this.props

    return (
      <Overlay visible={true}>
        <Overlay.Container maxWidth={600}>
          <Overlay.Heading title="Create Scraper" onDismiss={this.onDismiss} />
          <Form onSubmit={this.handleSubmit}>
            <Overlay.Body>
              <h5 className="wizard-step--sub-title">
                Scrapers collect data from multiple targets at regular intervals
                and to write to a bucket
              </h5>
              <CreateScraperForm
                buckets={buckets}
                url={scraper.url}
                name={scraper.name}
                selectedBucketID={scraper.bucketID}
                onInputChange={this.handleInputChange}
                onSelectBucket={this.handleSelectBucket}
              />
            </Overlay.Body>
            <Overlay.Footer>
              <Button
                text="Cancel"
                onClick={this.onDismiss}
                testID="create-scraper--cancel"
              />
              <Button
                status={this.submitButtonStatus}
                text="Create"
                onClick={this.handleSubmit}
                color={ComponentColor.Success}
                testID="create-scraper--submit"
              />
            </Overlay.Footer>
          </Form>
        </Overlay.Container>
      </Overlay>
    )
  }

  private handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value
    const key = e.target.name
    const scraper = {...this.state.scraper, [key]: value}

    this.setState({
      scraper,
    })
  }

  private handleSelectBucket = (bucket: Bucket) => {
    const {organizationID, id} = bucket
    const scraper = {...this.state.scraper, orgID: organizationID, bucketID: id}

    this.setState({scraper})
  }

  private get submitButtonStatus(): ComponentStatus {
    const {scraper} = this.state

    if (!scraper.url || !scraper.name || !scraper.bucketID) {
      return ComponentStatus.Disabled
    }

    return ComponentStatus.Default
  }

  // TODO: MOVE TO ACTION THUNK
  private handleSubmit = async () => {
    try {
      const {onCreateScraper, notify} = this.props
      const {scraper} = this.state

      await onCreateScraper(scraper)
      this.onDismiss()
      notify(scraperCreateSuccess())
    } catch (e) {
      console.error(e)
      notify(scraperCreateFailed())
    }
  }

  private get origin(): string {
    return window.location.origin
  }

  private onDismiss = (): void => {
    const {
      router,
      params: {orgID},
    } = this.props
    router.push(`/orgs/${orgID}/scrapers`)
  }
}

const mstp = ({buckets}: AppState): StateProps => ({
  buckets: buckets.list,
})

const mdtp: DispatchProps = {
  notify: notifyAction,
  onCreateScraper: createScraper,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(withRouter<StateProps & DispatchProps & OwnProps>(CreateScraperOverlay))
